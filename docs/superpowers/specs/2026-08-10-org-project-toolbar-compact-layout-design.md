# Organization Project Toolbar Compact Layout

## Context

The organization project ledger currently renders its search and filter controls as one wrapping flex container. After the project dropdowns were unified with the wider form-control style, each control can consume a full row and the toolbar becomes taller than the project list requires.

This change only tightens the ledger toolbar layout. Search behavior, filter values, URL query parameters, permissions, project data, and result rendering remain unchanged.

## Approved Layout

The desktop toolbar uses exactly two visual rows.

The first row contains, from left to right:

1. Project search.
2. Owner.
3. Risk.
4. Stage.
5. Target date.

The search control receives roughly twice the width of one filter control. The four filter controls share the remaining width equally. The search control must not occupy a row by itself at the current desktop ledger width.

The second row contains:

- “Only mine” and “Include archived projects” on the left.
- The existing filter submit button on the right.
- The existing clear-filters action next to the submit button when filters are active.

The toolbar remains one GET form. Pressing Enter in the search field or activating the submit button sends all search and filter values together. No second search button or client-side filtering behavior is introduced.

## Responsive Behavior

At the existing desktop ledger width, the first row remains a five-column grid and the second row remains a compact action row.

On narrower screens, the controls may wrap to preserve readable labels and usable target sizes:

- The search control spans the available width.
- Filter controls collapse to two columns, then one column on the narrowest screens.
- Checkboxes and actions wrap without horizontal page overflow.

Existing accessible labels, native form semantics, keyboard behavior, focus treatment, and minimum control height remain intact.

## Implementation Boundaries

- Update `templates/orgproject/shared/ledger.tmpl` only to add structural wrappers for the two rows. Preserve control order, names, values, labels, and query parameters.
- Update `web_src/css/features/org-project.css` to use a five-column grid for the first row and a spaced flex layout for the second row.
- Reuse the existing organization-project spacing, border, radius, color, and control-height tokens.
- Do not change project search logic, filter parsing, translations, list queries, table rendering, or empty states.
- Do not add a second submit action, automatic submission, or JavaScript-specific layout behavior.

## Verification

- Confirm the desktop toolbar renders as two visual rows at the width shown in the supplied ledger screenshot.
- Confirm the first row order is search, owner, risk, stage, and target date.
- Confirm both checkboxes and the submit button remain on the second row.
- Confirm the clear-filters action appears on the second row only when filters are active.
- Confirm Enter in the search field and the submit button preserve the existing GET query behavior.
- Confirm narrow layouts do not create horizontal page overflow.
- Run template formatting, CSS linting, and the focused organization-project tests affected by the template change.
