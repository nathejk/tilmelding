package query

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/nathejk/shared-go/types"
	"nathejk.dk/table"
)

type query struct {
	patrulje          *sql.Stmt
	spejdere          *sql.Stmt
	spejder           *sql.Stmt
	klan              *sql.Stmt
	seniorer          *sql.Stmt
	senior            *sql.Stmt
	isOpen            *sql.Stmt
	maxSeatCount      *sql.Stmt
	patruljeSeatCount *sql.Stmt
	klanSeatCount     *sql.Stmt
	startDate         *sql.Stmt
	pincode           *sql.Stmt
	mailTemplate      *sql.Stmt
}

func logFatalError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func New(db *sql.DB) *query {
	prepare := func(query string) *sql.Stmt {
		stmt, err := db.Prepare(query)
		if err != nil {
			log.Fatal(err)
		}
		return stmt
	}
	q := &query{}
	q.patrulje = prepare("SELECT teamId, name, groupName, korps, contactName, contactPhone, contactEmail, contactRole, signupStatus FROM patrulje WHERE teamId = ?")
	q.spejdere = prepare("SELECT memberId, teamId, name, address, postalCode, city, email, phone, phoneParent, birthday, `returning` FROM spejder WHERE teamId = ? ORDER BY createdAt")
	q.spejder = prepare("SELECT memberId, teamId, name, address, postalCode, city, email, phone, phoneParent, birthday, `returning` FROM spejder WHERE memberId = ?")
	q.klan = prepare("SELECT teamId, name, groupName, korps, signupStatus FROM klan WHERE teamId = ?")
	q.seniorer = prepare("SELECT memberId, teamId, name, address, postalCode, city, email, phone, birthday, `returning` FROM senior WHERE teamId = ? ORDER BY createdAt")
	q.senior = prepare("SELECT memberId, teamId, name, address, postalCode, city, email, phone, birthday, `returning` FROM senior WHERE memberId = ?")

	q.isOpen = prepare("SELECT isOpen FROM signup WHERE teamType = ?")
	q.maxSeatCount = prepare("SELECT maxSeatCount FROM signup WHERE teamType = ?")
	q.patruljeSeatCount = prepare("SELECT count(*) FROM patrulje WHERE signupStatus IN ('PAY', 'PAID')")
	q.klanSeatCount = prepare("SELECT count(*) FROM senior s JOIN klan k ON s.teamId = k.teamId WHERE k.signupStatus IN ('PAY', 'PAID')")
	q.startDate = prepare("SELECT startDate FROM signup WHERE teamType = ?")
	q.pincode = prepare("SELECT pincode FROM pincode WHERE teamID = ?")
	q.mailTemplate = prepare("SELECT subject, template FROM mailTemplate WHERE slug = ?")

	return q
}

func (q *query) Patruljer() []table.Patrulje {
	return nil
}

func (q *query) Patrulje(ID types.TeamID) (*table.Patrulje, error) {
	var p table.Patrulje
	err := q.patrulje.QueryRow(ID).Scan(&p.TeamID, &p.Name, &p.GroupName, &p.Korps, &p.ContactName, &p.ContactPhone, &p.ContactEmail, &p.ContactRole, &p.SignupStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}
func (q *query) Spejdere(ID types.TeamID) (spejdere []table.Spejder, e error) {
	rows, err := q.spejdere.Query(ID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var t table.Spejder
		err := rows.Scan(&t.MemberID, &t.TeamID, &t.Name, &t.Address, &t.PostalCode, &t.City, &t.Email, &t.Phone, &t.PhoneParent, &t.Birthday, &t.Returning)
		if err != nil {
			return nil, err
		}
		spejdere = append(spejdere, t)
	}
	return spejdere, nil
}
func (q *query) Spejder(ID types.MemberID) (*table.Spejder, error) {
	var t table.Spejder
	err := q.spejder.QueryRow(ID).Scan(&t.MemberID, &t.TeamID, &t.Name, &t.Address, &t.PostalCode, &t.City, &t.Email, &t.Phone, &t.PhoneParent, &t.Birthday, &t.Returning)
	if err != nil {
		return nil, err
	}
	return &t, nil
	/*
		var p table.Spejder
		err := q.Spejder.QueryRow(ID).Scan(&p.TeamID, &p.Name, &p.GroupName, &p.Korps, &p.ContactName, &p.ContactPhone, &p.ContactEmail, &p.ContactRole)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, nil
			}
			return nil, err
		}
		return &p, nil
	*/
}
func (q *query) Klan(ID types.TeamID) (*table.Klan, error) {
	var p table.Klan
	err := q.klan.QueryRow(ID).Scan(&p.TeamID, &p.Name, &p.GroupName, &p.Korps, &p.SignupStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}
func (q *query) Seniorer(ID types.TeamID) (seniorer []table.Senior, e error) {
	rows, err := q.seniorer.Query(ID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var t table.Senior
		err := rows.Scan(&t.MemberID, &t.TeamID, &t.Name, &t.Address, &t.PostalCode, &t.City, &t.Email, &t.Phone, &t.Birthday, &t.Returning)
		if err != nil {
			return nil, err
		}
		seniorer = append(seniorer, t)
	}
	return seniorer, nil
}
func (q *query) Senior(ID types.MemberID) (*table.Senior, error) {
	var t table.Senior
	err := q.senior.QueryRow(ID).Scan(&t.MemberID, &t.TeamID, &t.Name, &t.Address, &t.PostalCode, &t.City, &t.Email, &t.Phone, &t.Birthday, &t.Returning)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (q *query) TeamType(teamID types.TeamID) types.TeamType {
	if patrulje, _ := q.Patrulje(teamID); patrulje != nil {
		return types.TeamTypePatrulje
	}
	if klan, _ := q.Klan(teamID); klan != nil {
		return types.TeamTypeKlan
	}
	return ""
}
func (q *query) IsOpen(teamType types.TeamType) bool {
	var v bool
	if err := q.isOpen.QueryRow(teamType).Scan(&v); err != nil {
		log.Printf("Error in IsOpen(%s) %s", teamType, err)
		return false
	}
	return v
}
func (q *query) MaxSeatCount(teamType types.TeamType) int {
	var v int
	if err := q.maxSeatCount.QueryRow(teamType).Scan(&v); err != nil {
		return 0
	}
	return v
}
func (q *query) UsedSeatCount(teamType types.TeamType) int {
	var v int
	switch teamType {
	case types.TeamTypePatrulje:
		if err := q.patruljeSeatCount.QueryRow().Scan(&v); err != nil {
			return 0
		}
	case types.TeamTypeKlan:
		if err := q.klanSeatCount.QueryRow().Scan(&v); err != nil {
			return 0
		}
	}
	return v
}
func (q *query) SignupStart(teamType types.TeamType) *time.Time {
	var v string
	if err := q.startDate.QueryRow(teamType).Scan(&v); err != nil {
		return nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil
	}
	return &t
}
func (q *query) SignupStarted(teamType types.TeamType) bool {
	t := q.SignupStart(types.TeamTypeKlan)
	return t == nil || t.Before(time.Now())
}
func (q *query) Pincode(teamID types.TeamID) string {
	var v string
	if err := q.pincode.QueryRow(teamID).Scan(&v); err != nil {
		return ""
	}
	return v
}

func (q *query) MailTemplate(slug types.Slug) (string, string) {
	var subj, tmpl string
	if err := q.mailTemplate.QueryRow(slug).Scan(&subj, &tmpl); err != nil {
		return "", ""
	}
	return subj, tmpl
}
func (q *query) MailTemplateData(teamID types.TeamID) types.MailTemplateData {
	data := types.MailTemplateData{
		Nathejk: fmt.Sprintf("Nathejk %d", time.Now().Year()),
		Weekend: "17. - 19. September",
	}
	switch q.TeamType(teamID) {
	case types.TeamTypePatrulje:
		t, _ := q.Patrulje(teamID)
		data.Name = t.Name
		data.Group = t.GroupName
		data.Corps = t.Korps
		data.SignupStatus = t.SignupStatus
		data.Contact = types.MailTemplateData_Contact{
			Name:  t.ContactName,
			Phone: t.ContactPhone,
			Email: t.ContactEmail,
			Role:  t.ContactRole,
		}
		spejdere, _ := q.Spejdere(teamID)
		for _, spejder := range spejdere {
			data.Members = append(data.Members, types.MailTemplateData_Member{
				Name:        spejder.Name,
				Address:     spejder.Address,
				PostalCode:  spejder.PostalCode,
				City:        spejder.City,
				Email:       spejder.Email,
				Phone:       spejder.Phone,
				PhoneParent: spejder.PhoneParent,
				Birthday:    spejder.Birthday,
				Returning:   spejder.Returning,
			})
		}

	case types.TeamTypeKlan:
	default:
	}
	return data
}
