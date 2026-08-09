# Organization Project Create Page Redesign

## Summary

Redesign the organization project create page as a guided workbench. The page keeps the existing Gitea organization navigation, native form submission, configured project-field schema, permission checks, and server-side error handling. The change reorganizes the form into clear sections, adds a read-only creation checklist, and introduces responsive behavior that removes the excessive whitespace and weak hierarchy in the current page.

The approved direction is option A, **Guided Workbench**.

## Goals

- Make the form easy to scan before users begin entering data.
- Group fields by user intent instead of presenting one long undifferentiated grid.
- Keep the primary action visible without duplicating it.
- Provide a compact, read-only summary of important creation choices.
- Preserve configured custom fields and the current native POST contract.
- Work without horizontal scrolling at 320, 375, 414, and 768 pixels.
- Match the existing organization project ledger and detail-page design system.

## Non-goals

- No route, permission, model, schema, or persistence changes.
- No changes to repository project boards under repository routes.
- No wizard, autosave, modal, or multi-step submission flow.
- No redesign of the organization project detail edit form.
- No new configuration concept for assigning custom fields to sections.
- No changes to the create button's submission semantics.

## Design System

- Genre: modern-minimal developer tool.
- Macrostructure: Guided Workbench, derived from the existing Workbench family.
- Theme: Gitea native through the existing project tokens and Gitea theme variables.
- Typography: existing Gitea sans and monospace families; headings remain upright.
- Spacing: existing four-pixel project scale.
- Enrichment: none; the form is the page content.
- Motion: no entrance animation. Only existing control feedback is retained.
- Navigation: inherit the global and organization project navigation.

## Page Structure

### Heading

The page starts with a compact breadcrumb, title, and existing description. The action group sits at the upper right:

- Secondary action: Cancel.
- Primary action: Create project.

The primary action appears once. It stays visible near the page heading on desktop and moves below the heading copy on narrow screens. It is not duplicated in the sidebar or at the form bottom.

### Desktop layout

At wide widths, the form uses a two-column workbench:

- Main column: three stacked form panels plus an optional fourth panel.
- Sidebar: a sticky, read-only creation checklist.
- Main-to-sidebar ratio: approximately 1.8:0.68 with the sidebar capped near 18rem.

The main column contains:

1. **Basic information**
   - Name
   - Slug
   - Description
2. **Plan and responsibility**
   - Stage
   - Owner
   - Followers
   - Start date
   - Target date
3. **Status and action**
   - Progress
   - Risk
   - Summary
   - Current problem
   - Next action
   - Next action owner
   - Next action due date
4. **Other information**, rendered only when the active schema contains fields not recognized by the three groups above.

Long text, multi-select, and member-array controls span the full main column. Other fields use a two-column grid.

### Creation checklist

The sidebar summarizes these important values when they exist in the active form:

- Slug
- Stage
- Followers
- Risk
- Owner
- Target date

Each row includes a text status in addition to its visual marker. Empty optional values display a neutral "Not set" state. The checklist is informational and never disables or intercepts form submission.

The checklist listens to native `input` and `change` events through event delegation on the create form. It reads current native input and select values, including the controls rendered by Vue. It does not own form state and does not write values back into controls.

### Responsive order

Below the workbench breakpoint, the page becomes a single column in this order:

1. Heading and actions
2. Basic information
3. Plan and responsibility
4. Status and action
5. Other information, when present
6. Creation checklist

The sidebar stops being sticky. All form grids collapse to one column. Buttons stay on one line and expand only when the available width requires it.

## Component Boundaries

### Server template

`templates/orgproject/new.tmpl` continues to own:

- The native `form` element and POST action.
- CSRF markup.
- Name, slug, and description controls.
- Error message rendering.
- Heading, action group, main workbench wrapper, and checklist markup.
- Localized section and checklist labels exposed through template markup or data attributes.

The template marks the create form with a dedicated data attribute so the summary enhancer cannot run on the detail edit form.

### Configured-field form

`ProjectForm.vue` gains an explicit presentation mode:

- `plain`, the default, preserves the current detail edit form.
- `grouped`, used only by the create page.

In grouped mode, fields are partitioned by known default-schema keys. Unknown active fields fall into the Other information group in their existing schema order. The hidden `values` JSON input and all initialization and serialization behavior remain unchanged.

To avoid duplicating the rendering logic for every supported field type, a focused `ProjectFormField.vue` component renders one configured field. It receives the field definition, current value, and members, then updates the parent value through `v-model`. It does not serialize or submit data itself.

### Checklist enhancer

`web_src/js/features/org-project.ts` initializes the create checklist after the existing Vue mounts complete. The enhancer:

- Uses one delegated listener on the create form.
- Resolves current labels for single- and multi-select controls.
- Updates only checklist text and state classes.
- Leaves the checklist static if its expected form control is absent.
- Does nothing when the create-page data attribute is absent.

The enhancer remains a progressive enhancement. A failure in this code cannot prevent the native form from submitting.

### Styling

`web_src/css/features/org-project.css` receives page-scoped styles for:

- The create-page heading and actions.
- Workbench grid and stacked form panels.
- Panel headers and field grids.
- Sticky checklist and its statuses.
- Narrow-screen ordering and one-column fields.

All colors, fonts, spacing, radii, and durations reference the existing project or Gitea tokens. No new hard-coded brand palette or global style override is introduced.

## Data Flow

1. The server renders the native form, old core values, configured schema, old configured values, and organization members.
2. `ProjectForm.vue` initializes configured values exactly as it does today and renders grouped or plain presentation according to its prop.
3. Vue continues to serialize configured values into the hidden `values` input.
4. The checklist enhancer reads visible control values after Vue mounting and on subsequent native form events.
5. The browser submits the same core fields and hidden `values` JSON to the existing POST route.
6. The server performs the existing validation and either re-renders the page with values and an error or redirects after successful creation.

No checklist state is submitted to the server.

## Validation and Error Handling

- Existing HTML `required` attributes remain the first validation layer.
- Browser validation continues to focus the first invalid native control.
- The create button remains enabled; the checklist does not create a second validation system.
- Existing `OrgProjectError` rendering stays above the form workbench.
- Re-rendered core and configured values repopulate both the form and checklist.
- Missing or unknown configured field keys render under Other information instead of being discarded.
- If JavaScript fails, core fields and native submission remain available. Configured fields retain the project's current JavaScript dependency and failure behavior.

## Accessibility

- Each panel has a visible heading and an `aria-labelledby` relationship where needed.
- Existing label-to-control associations and required semantics remain intact.
- Checklist markers are paired with visible text; status is never color-only.
- The checklist is not an `aria-live` region, avoiding noisy announcements on every keystroke.
- The page preserves native tab order and existing Gitea focus styles.
- Buttons and links keep concise, non-wrapping labels.
- The narrow layout has no fixed-width form controls or horizontal page scrolling.

## Testing

### Vue unit tests

Extend `ProjectForm.test.ts` to verify:

- Grouped mode assigns known keys to the approved sections.
- Unknown active fields render in Other information in schema order.
- Plain mode preserves the existing flat field layout.
- Every configured field type still updates the hidden `values` JSON correctly.

Add focused tests for `ProjectFormField.vue` only where field rendering behavior is not already covered through `ProjectForm.test.ts`.

### Feature unit tests

Add `web_src/js/features/org-project.test.ts` to verify:

- Checklist initialization reads existing core and mounted Vue controls.
- Text inputs, single selects, multi-selects, member selects, and dates update their summary rows.
- Missing controls remain in a neutral state without throwing.
- Pages without the create-form marker are ignored.

### End-to-end coverage

Extend `tests/e2e/org-project.test.ts` to verify:

- The grouped create page renders the approved section order.
- A project can still be created through the native form.
- Server-side errors remain visible with entered values preserved.
- The page has no horizontal overflow at the required mobile widths.

### Required checks

- Run the focused Vitest files.
- Run the focused organization-project Playwright file when the local E2E environment is available.
- Run `make lint-js`.
- Run `make lint-css`.
- Run `make lint-templates`.

## Expected File Changes

Modify:

- `templates/orgproject/new.tmpl`
- `web_src/js/components/orgproject/ProjectForm.vue`
- `web_src/js/components/orgproject/ProjectForm.test.ts`
- `web_src/js/features/org-project.ts`
- `web_src/css/features/org-project.css`
- `options/locale/locale_en-US.json`
- `options/locale/locale_zh-CN.json`
- `tests/e2e/org-project.test.ts`

Create:

- `web_src/js/components/orgproject/ProjectFormField.vue`
- `web_src/js/features/org-project.test.ts`

Delete:

- None.

## Acceptance Criteria

- The create page matches the approved Guided Workbench layout.
- Desktop users see grouped panels and a sticky read-only checklist.
- Narrow screens show one ordered column with the checklist after the form fields.
- The detail edit form retains its current plain layout.
- Custom configured fields remain visible, ordered, editable, and serialized.
- Existing successful creation, validation, permission, and error paths remain unchanged.
- The checklist accurately reflects supported current values but never controls submission.
- The page has no horizontal scrolling at 320, 375, 414, or 768 pixels.
- No production files are deleted.
