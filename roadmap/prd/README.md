# Product Requirements Documents

This directory holds Product Requirements Documents (PRDs) — the agreed
definition of *what* a feature is and *why* it exists, written before
implementation begins.

## Conventions

- One PRD per file, named with a zero-padded sequence and a slug:

  ```
  001-participant-self-service-signup.md
  002-team-leader-dashboard.md
  ```

- The sequence number is permanent. Check the highest existing number here
  before creating a new PRD.
- Use the standard template and authoring workflow provided by the `prd` agent
  skill (`.agents/skills/prd/`). Its `prd-template.md` is the canonical
  structure every PRD should follow.
- Dates use `YYYY-MM-DD`, matching the task board.

## Relationship to the task board

A PRD defines the problem, goals, and requirements. Execution is tracked
separately on the file-based task board in `roadmap/tasks/` (see
`roadmap/tasks/TASKS.md`). Each PRD's "Rollout / Task Breakdown" section
proposes the concrete tasks that should be created in `roadmap/tasks/open/`.

## Lifecycle

PRDs move through these statuses (tracked in the document header):

`draft` → `in-review` → `approved` → `shipped`

Keep PRDs in this directory after shipping — they are a useful record of intent
and decisions.
