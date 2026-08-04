# Product Requirements Documents

This directory holds Product Requirements Documents (PRDs) — the agreed
definition of *what* a feature is and *why* it exists, written before
implementation begins.

## Folder structure

```
roadmap/prd/
  draft/      ← being written; not yet agreed
  doing/      ← agreed and being implemented
  done/       ← shipped
  README.md   ← this file
```

A PRD's folder is its status. Moving the file between folders is how status
changes, exactly as on the task board (`roadmap/tasks/`), and it produces a
readable git diff.

## Conventions

- One PRD per file, named with a zero-padded sequence and a slug:

  ```
  001-participant-self-service-signup.md
  002-team-leader-dashboard.md
  ```

- The sequence number is permanent and never changes, even as the file moves
  between folders. Check the highest existing number **across all three
  folders** before creating a new PRD.
- Use the standard template and authoring workflow provided by the `prd` agent
  skill (`.agents/skills/prd/`). Its `prd-template.md` is the canonical
  structure every PRD should follow.
- Dates use `YYYY-MM-DD`, matching the task board.
- **Refer to a PRD by number, not by path.** Paths change as PRDs move between
  folders, so a link like `roadmap/prd/draft/003-foo.md` goes stale. Write
  "PRD 003" instead.

## Lifecycle

PRDs move `draft/` → `doing/` → `done/`. **The `Status` field in the document
header must always match the folder the file is in** — the two are a single
piece of information recorded twice, and a mismatch means the document is
lying about itself.

### draft/ → doing/

The approval gate. Move the file when the PRD has been reviewed and agreed and
implementation is starting.

- Set `Status: doing`
- Set `Approved` to today's date
- Update `Last updated`
- Create the tasks from the "Rollout / Task Breakdown" section in
  `roadmap/tasks/open/`

### doing/ → done/

Move the file when every task derived from the PRD is in
`roadmap/tasks/done/` and the feature is shipped.

- Set `Status: done`
- Set `Shipped` to today's date
- Update `Last updated`

### Keeping a PRD honest while it is in doing/

Requirements shift during implementation. When they do, edit the PRD and bump
`Last updated` — a PRD in `doing/` that no longer describes what is being
built is worse than no PRD. If the change is large enough to invalidate the
agreement, move the file back to `draft/` (reset `Status`, clear `Approved`)
rather than quietly rewriting an approved document.

PRDs stay in `done/` after shipping — they are a useful record of intent and
decisions.

Create a folder if it does not exist; git does not track empty directories, so
`draft/` and `doing/` may be absent when there is nothing in them.

## Relationship to the task board

A PRD defines the problem, goals, and requirements. Execution is tracked
separately on the file-based task board in `roadmap/tasks/` (see
`roadmap/tasks/TASKS.md`). Each PRD's "Rollout / Task Breakdown" section
proposes the concrete tasks that should be created in `roadmap/tasks/open/`.

The two boards move independently but must not contradict each other. In
particular, a PRD whose tasks are all `done` belongs in `prd/done/` — if it is
still sitting in `draft/`, the status was never updated.
