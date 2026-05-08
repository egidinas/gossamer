# Gossamer Backlog

Items in priority order. Each item has a status, description, and acceptance criteria.

---

## GOSS-01 · Marker label overlap
**Status:** open  
Dense test phases on the primary FAT/TVac card render diagonal phase labels that collide at normal zoom levels. Labels become unreadable at 4-cycle or 8-cycle density.  
**Fix:** render short labels (≤8 chars) by default; suppress overlapping neighbours; show full label on hover via tooltip.  
**AC:** no two visible labels overlap at 1440p full-zoom on thermal_acceptance_fat and tvac_qualification.

---

## GOSS-02 · 4K card height under-utilisation
**Status:** open  
At 3840px the operator center lanes are ~352px each, leaving large dead space below the fourth lane. The `clamp` ceiling is capped at 320px which was designed for 1080p headroom.  
**Fix:** raise the upper bound of the `clamp` for `command_center_fat` lanes at wide viewports so all four lanes together fill ~85% of the viewport height.  
**AC:** at 3840×2160 the four command-center lanes collectively occupy ≥80% of viewport height.

---

## GOSS-03 · functional_events card unbounded height
**Status:** open  
The `functional_events` event-rail card grows to 1271–1313px on desktop because the event rail has no height cap. It dwarfs every other card and breaks section rhythm.  
**Fix:** cap event-rail cards at `max-height: 480px` with `overflow-y: auto` inside the plot shell; or cap the swimlane row count at render time.  
**AC:** functional_events card ≤ 500px at 1440p; content scrollable if truncated.

---

## GOSS-04 · Mobile graph horizontal scroll
**Status:** open  
Acceptance FAT and Qualification TVac overflow by ~243px on mobile (390px viewport). The time axis and right-side y-labels are clipped. Currently noted as acceptable but a scroll container would make the graph pannable.  
**Fix:** wrap `.operator-wall-scrollframe` content in an `overflow-x: auto` scroll container at narrow viewports; or constrain the shared time axis and label-rail to the visible width.  
**AC:** on 390px viewport, acceptance and tvac graphs are either fully contained or explicitly horizontally scrollable with no clipping.

---

## GOSS-05 · Remove one-off scripts from repo root
**Status:** open  
`do_gofmt.sh`, `do_refactor.py`, `refactor_*.py`, `fix_recover.py`, `safego_refactor.py`, and similar one-shot files are sitting untracked in the repo root. They pollute `git status` and are confusing to anyone cloning the repo.  
**Fix:** delete all of them (they were single-use refactor aids, not ongoing tools).  
**AC:** `git status` shows no untracked `*.py` or ad-hoc `*.sh` files in the repo root.

---

## GOSS-06 · Commit outstanding fixture changes
**Status:** open  
`thermal_acceptance_fat` and `tvac_qualification` tiles, manifests, and telemetry archives are modified but not committed. The working tree is dirty against the deployed state.  
**Fix:** regenerate fixtures cleanly (`go run ./cmd/gossamer-fixtures`) and commit the result, or stage and commit the current modified files if they are already the intended state.  
**AC:** `git status` shows no modified fixture files; deployed bundle matches HEAD.
