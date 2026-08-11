# Organization Project History Timeline Design

## Goal

Replace the raw `source · request_id` list with a readable project audit timeline without changing the persisted audit model or public API.

## Data flow

The Web handler continues to read the newest 100 `org_project_change_log` rows for the current project. It converts those rows into a Web-only JSON DTO with stable lower-case fields, parsed `changed_fields`, `before`, and `after` JSON, actor display information, source, request ID, and creation time. Actor records are loaded in one batch; missing actors fall back to their numeric ID.

The current published project schema supplies labels for dynamic `values.<key>` fields. Unknown or removed fields fall back to their stable key.

## UI

Each entry shows:

- actor display name, linked when the user still exists;
- a readable list of changed fields;
- relative creation time and localized source label;
- an expandable detail area with one row per changed field and explicit “Before” and “After” values;
- the request ID as secondary diagnostic metadata rather than the primary title.

The diff becomes one column on narrow screens. Empty values use an em dash. Structured values remain formatted JSON so repository and multi-value changes stay inspectable.

## Scope

- Keep the existing audit table and API response unchanged.
- Do not add pagination; the Web page remains limited to 100 newest records.
- Do not attempt to reconstruct deleted historical schema labels or repository names.
- Add focused rendering and Web DTO tests, then run formatting and targeted lint/tests.

## Success criteria

1. History entries no longer render as only `web · UUID`.
2. `BeforeValue` and `AfterValue` are correctly delivered and displayed.
3. Actor, time, source, changed fields, and request ID are visible.
4. Dynamic field labels use the current published schema with a key fallback.
5. The layout remains usable on mobile and relevant tests/lints pass.
