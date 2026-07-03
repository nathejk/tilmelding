# 008 — Remove commented-out createSignupHandler block

**Status:** open
**Priority:** low
**Created:** 2026-06-04

## Description

`cmd/api/signup.go` contains a long `/* … */` block of pre-cutover code that was never deleted. It includes a stale `app.mailer.Send(...)` call plus the legacy "build a verify-email message" path now lived by `signup.commander.SendVerificationEmail`.

Decision: remove the block entirely (the verify-email flow now lives in `nathejk/table/signup/commands.go`), or revive it using the order flow if there's a missing handler that was intended.

Related files:
- `go/cmd/api/signup.go`

## Acceptance Criteria

- [ ] Commented-out block removed (or revived with working code if needed)
- [ ] No stale references to old mailer.Send pattern remain

## Progress Log

- 2026-06-04 21:54 — Task created.
