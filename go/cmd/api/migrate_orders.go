package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
	jsonapi "nathejk.dk/cmd/api/app"
)

// Product prices in øre, matching product.Seeds2026. Hard-coded so the
// migration is self-contained — running it doesn't depend on the
// catalogue projection being current at migration time.
const (
	migratePriceParticipationPatrulje = 25000
	migratePriceParticipationKlan     = 25000
	migratePriceParticipationCrew     = 10000
	migratePriceParticipationGogler   = 10000
	migratePriceTShirt                = 17500
)

// migrateLegacyOrdersHandler backfills synthetic paid orders for teams
// that paid through the legacy payment flow (payment.orderForeignKey =
// teamId, orderType != 'order') but don't yet have a paid order in the
// new order system. After running, PaidLineKeys correctly reports what
// each team has already paid for and the open order won't double-bill.
//
// The handler is idempotent: teams that already have a paid order are
// skipped. Calling it twice is safe.
//
// Auth: requires the X-Migrate-Token request header to match the
// MIGRATE_TOKEN environment variable. If MIGRATE_TOKEN is unset the
// handler refuses every request — this avoids exposing the endpoint by
// accident in environments where the env var hasn't been set.
//
// Query params:
//
//	year=2026   (optional; defaults to the API's configured year)
//	dry=1       (optional; report what would be published without publishing)
//
// Response: JSON envelope with "found", "migrated", and "results" keys.
func (app *application) migrateLegacyOrdersHandler(w http.ResponseWriter, r *http.Request) {
	expected := os.Getenv("MIGRATE_TOKEN")
	if expected == "" {
		app.NotFoundResponse(w, r)
		return
	}
	if r.Header.Get("X-Migrate-Token") != expected {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	year := r.URL.Query().Get("year")
	if year == "" {
		year = string(app.config.year)
	}
	dryRun := r.URL.Query().Get("dry") == "1"

	report, err := runLegacyOrderMigration(r.Context(), app.db, app.publisher, year, dryRun)
	if err != nil {
		app.logger.PrintError(err, map[string]string{"year": year})
		app.ServerErrorResponse(w, r, err)
		return
	}

	if err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{
		"year":     year,
		"dryRun":   dryRun,
		"found":    report.Found,
		"migrated": report.Migrated,
		"results":  report.Results,
	}, nil); err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

// migrationReport captures per-team outcomes for the JSON response.
type migrationReport struct {
	Found    int
	Migrated int
	Results  []migrationResult
}

type migrationResult struct {
	OwnerType   string `json:"ownerType"`
	OwnerID     string `json:"ownerId"`
	OrderID     string `json:"orderId,omitempty"`
	LineCount   int    `json:"lineCount"`
	TotalAmount int    `json:"totalAmount"` // øre
	PaidAmount  int    `json:"paidAmount"`  // øre, what the legacy payment(s) recorded
	Status      string `json:"status"`      // "ok" | "skipped" | "error"
	Reason      string `json:"reason,omitempty"`
}

// legacyTeam carries one row from findLegacyTeams — a team with legacy
// payments that needs a synthetic paid order.
type legacyTeam struct {
	teamID    string
	ownerType types.TeamType
	paidOre   int
}

// memberLine is a (memberId, tshirtSize) pair sourced from the
// spejder/senior/personnel projection at migration time.
type memberLine struct {
	memberID   string
	tshirtSize string
}

// runLegacyOrderMigration is the migration entry point. It finds every
// team with legacy payments and no paid order, builds derived lines from
// the current member projections, truncates lines to the paid budget when
// the team underpaid, and publishes the three order events per team.
func runLegacyOrderMigration(ctx context.Context, db *sql.DB, publisher stream.Publisher, year string, dryRun bool) (migrationReport, error) {
	teams, err := findLegacyTeams(ctx, db, year)
	if err != nil {
		return migrationReport{}, fmt.Errorf("findLegacyTeams: %w", err)
	}

	report := migrationReport{Found: len(teams), Results: make([]migrationResult, 0, len(teams))}

	for _, team := range teams {
		res := migrationResult{
			OwnerType:  string(team.ownerType),
			OwnerID:    team.teamID,
			PaidAmount: team.paidOre,
		}

		lines, totalAmount := buildLinesForTeam(ctx, db, team, year)
		if len(lines) == 0 {
			res.Status = "skipped"
			res.Reason = "no members found"
			report.Results = append(report.Results, res)
			continue
		}

		if totalAmount > team.paidOre {
			lines, totalAmount = truncateLinesToBudget(lines, team.paidOre)
		}
		if len(lines) == 0 || totalAmount == 0 {
			res.Status = "skipped"
			res.Reason = "paid amount does not cover any line"
			report.Results = append(report.Results, res)
			continue
		}

		orderID := uuid.NewString()
		now := time.Now()
		res.OrderID = orderID
		res.LineCount = len(lines)
		res.TotalAmount = totalAmount

		if dryRun {
			res.Status = "ok"
			res.Reason = "dry-run"
			report.Results = append(report.Results, res)
			report.Migrated++
			continue
		}

		if err := publishMigratedOrder(publisher, year, orderID, team, lines, totalAmount, now); err != nil {
			res.Status = "error"
			res.Reason = err.Error()
			report.Results = append(report.Results, res)
			continue
		}

		res.Status = "ok"
		report.Results = append(report.Results, res)
		report.Migrated++
	}

	return report, nil
}

// findLegacyTeams returns teams that paid via the legacy flow
// (orderType != 'order') and don't already have a paid order in the new
// system. Status filter matches AmountPaidByTeamID's contract — both
// 'reserved' and 'received' count toward the paid total.
func findLegacyTeams(ctx context.Context, db *sql.DB, year string) ([]legacyTeam, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT p.orderForeignKey, p.orderType, SUM(p.amount) AS totalPaid
		 FROM payment p
		 WHERE p.status IN ('reserved', 'received')
		   AND p.orderType != 'order'
		   AND p.year = ?
		   AND p.orderForeignKey NOT IN (
		       SELECT o.ownerId FROM orders o WHERE o.year = ? AND o.status = 'paid'
		   )
		 GROUP BY p.orderForeignKey, p.orderType
		 HAVING totalPaid > 0`,
		year, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []legacyTeam
	for rows.Next() {
		var t legacyTeam
		var orderType string
		if err := rows.Scan(&t.teamID, &orderType, &t.paidOre); err != nil {
			return nil, err
		}
		t.ownerType = types.TeamType(orderType)
		teams = append(teams, t)
	}
	return teams, rows.Err()
}

// buildLinesForTeam constructs participation + optional t-shirt lines
// for every current member of the team. Mirrors the derivedLinesFor*
// pattern in the patrulje / klan / personnel handlers, including the
// "derived:<sku>:<memberId>" lineId convention so SetDerivedLines later
// dedupes against these via PaidLineKeys.
//
// Personnel signups (crew / gøgler) also flow through this: the
// "team" is one person and the personnel projection is queried with
// userId == ownerId. Legacy orderType values ("staff", "friend") are
// honoured here because the rename's DB migration may not have run
// against existing payment rows yet — they all map onto crew.
func buildLinesForTeam(ctx context.Context, db *sql.DB, team legacyTeam, year string) ([]messages.NathejkOrder_Line, int) {
	var members []memberLine

	switch team.ownerType {
	case types.TeamTypePatrulje:
		members = queryMigrateSpejdere(ctx, db, team.teamID, year)
	case types.TeamTypeKlan:
		members = queryMigrateSeniore(ctx, db, team.teamID)
	case types.TeamTypeBadut, types.TeamTypeCrew, "staff", "friend":
		members = queryMigratePersonnel(ctx, db, team.teamID)
	default:
		return nil, 0
	}

	sku, name, price := migrateProductForType(team.ownerType)
	var lines []messages.NathejkOrder_Line
	total := 0

	for _, m := range members {
		lines = append(lines, messages.NathejkOrder_Line{
			LineID:      fmt.Sprintf("derived:%s:%s", sku, m.memberID),
			ProductSKU:  sku,
			ProductName: name,
			MemberID:    m.memberID,
			UnitPrice:   price,
			Quantity:    1,
			LineTotal:   price,
			Origin:      messages.LineOriginDerived,
		})
		total += price

		if m.tshirtSize != "" {
			lines = append(lines, messages.NathejkOrder_Line{
				LineID:      fmt.Sprintf("derived:tshirt.adult:%s", m.memberID),
				ProductSKU:  "tshirt.adult",
				ProductName: "T-shirt",
				MemberID:    m.memberID,
				UnitPrice:   migratePriceTShirt,
				Quantity:    1,
				LineTotal:   migratePriceTShirt,
				Origin:      messages.LineOriginDerived,
				Attributes:  map[string]any{"size": m.tshirtSize},
			})
			total += migratePriceTShirt
		}
	}
	return lines, total
}

// truncateLinesToBudget keeps as many lines as the paid amount covers
// for partially-paid teams. Two-pass: participation lines first (every
// member who got their seat), then t-shirts for those same members.
// Members whose participation didn't fit are dropped entirely so we
// don't claim "this member is paid" without their seat being covered.
func truncateLinesToBudget(lines []messages.NathejkOrder_Line, budget int) ([]messages.NathejkOrder_Line, int) {
	var kept []messages.NathejkOrder_Line
	total := 0
	paidMembers := map[string]bool{}

	for _, l := range lines {
		if l.ProductSKU == "tshirt.adult" {
			continue
		}
		if total+l.LineTotal > budget {
			break
		}
		kept = append(kept, l)
		total += l.LineTotal
		paidMembers[l.MemberID] = true
	}

	for _, l := range lines {
		if l.ProductSKU != "tshirt.adult" {
			continue
		}
		if !paidMembers[l.MemberID] {
			continue
		}
		if total+l.LineTotal > budget {
			break
		}
		kept = append(kept, l)
		total += l.LineTotal
	}
	return kept, total
}

// publishMigratedOrder fires the three events (created → lines.changed →
// paid) the order projector and saga need to materialise the synthetic
// order. It publishes through the shared metatagger publisher but overrides
// the producer tag with "migrate-orders" so the trail is visible in the
// stream, while still inheriting the tagger's other defaults (e.g. version).
func publishMigratedOrder(p stream.Publisher, year string, orderID string, team legacyTeam, lines []messages.NathejkOrder_Line, totalAmount int, now time.Time) error {
	meta := &messages.Metadata{Producer: "migrate-orders"}

	created := &messages.NathejkOrderCreated{
		OrderID:   orderID,
		Year:      types.YearSlug(year),
		OwnerType: normaliseOrderOwnerType(team.ownerType),
		OwnerID:   team.teamID,
		Currency:  "DKK",
		Timestamp: now,
	}
	msg := p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.order.%s.created", year, orderID)))
	msg.SetBody(created)
	msg.SetMeta(meta)
	if err := p.Publish(msg); err != nil {
		return fmt.Errorf("publish created: %w", err)
	}

	linesChanged := &messages.NathejkOrderLinesChanged{
		OrderID:     orderID,
		Lines:       lines,
		TotalAmount: totalAmount,
		Timestamp:   now,
	}
	msg = p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.order.%s.lines.changed", year, orderID)))
	msg.SetBody(linesChanged)
	msg.SetMeta(meta)
	if err := p.Publish(msg); err != nil {
		return fmt.Errorf("publish lines.changed: %w", err)
	}

	paid := &messages.NathejkOrderPaid{
		OrderID:    orderID,
		PaidAmount: totalAmount,
		Timestamp:  now,
	}
	msg = p.MessageFunc()(subject.FromStr(fmt.Sprintf("NATHEJK:%s.order.%s.paid", year, orderID)))
	msg.SetBody(paid)
	msg.SetMeta(meta)
	if err := p.Publish(msg); err != nil {
		return fmt.Errorf("publish paid: %w", err)
	}
	return nil
}

func migrateProductForType(ownerType types.TeamType) (sku, name string, price int) {
	switch ownerType {
	case types.TeamTypePatrulje:
		return "participation.patrulje", "Patrulje-deltagelse", migratePriceParticipationPatrulje
	case types.TeamTypeKlan:
		return "participation.klan", "Senior-deltagelse", migratePriceParticipationKlan
	case types.TeamTypeCrew, "staff", "friend":
		return "participation.crew", "Crew-deltagelse", migratePriceParticipationCrew
	case types.TeamTypeBadut:
		return "participation.gogler", "Gøgler-deltagelse", migratePriceParticipationGogler
	default:
		return "participation.crew", "Crew-deltagelse", migratePriceParticipationCrew
	}
}

// normaliseOrderOwnerType collapses pre-rename owner types onto their
// post-rename equivalents so the order entity only ever sees the four
// canonical owner types (patrulje, klan, crew, badut). Keep in sync
// with personnelOrderOwnerType in personnel.go — the runtime handlers
// use that one, the migration uses this one, and they must agree on
// the mapping or the runtime won't find migrated orders via
// FindOpenOrder.
func normaliseOrderOwnerType(t types.TeamType) types.TeamType {
	switch t {
	case "staff", "friend":
		return types.TeamTypeCrew
	}
	return t
}

func queryMigrateSpejdere(ctx context.Context, db *sql.DB, teamID, year string) []memberLine {
	rows, err := db.QueryContext(ctx,
		`SELECT s.memberId, s.tshirtSize FROM spejder s WHERE s.teamId = ? AND s.year = ?`,
		teamID, year)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanMembers(rows)
}

func queryMigrateSeniore(ctx context.Context, db *sql.DB, teamID string) []memberLine {
	rows, err := db.QueryContext(ctx,
		`SELECT s.memberId, s.tshirtSize FROM senior s WHERE s.teamId = ?`,
		teamID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanMembers(rows)
}

func queryMigratePersonnel(ctx context.Context, db *sql.DB, userID string) []memberLine {
	rows, err := db.QueryContext(ctx,
		`SELECT p.userId, p.tshirtSize FROM personnel p WHERE p.userId = ?`,
		userID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanMembers(rows)
}

func scanMembers(rows *sql.Rows) []memberLine {
	var out []memberLine
	for rows.Next() {
		var m memberLine
		if err := rows.Scan(&m.memberID, &m.tshirtSize); err != nil {
			continue
		}
		out = append(out, m)
	}
	return out
}
