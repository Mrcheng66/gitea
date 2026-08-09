# Organization Project Config Clone Fix

## Problem

`ConfigEditor` stores its schema in a Vue `ref`, which exposes the schema as a reactive `Proxy`. The `normalized` computed value passes that proxy to `normalizeSchema`, whose `cloneJSON` helper uses `structuredClone`. Browsers cannot structured-clone proxies, so opening the configuration editor raises `DataCloneError` and interrupts rendering.

## Design

Keep `cloneJSON` and `normalizeSchema` independent of Vue. In `ConfigEditor`, unwrap `schema.value` with Vue's `toRaw` immediately before passing it to `normalizeSchema`. This is the narrowest boundary where a Vue proxy enters the JSON-only normalization code, and it preserves the existing structured-clone behavior for all other callers.

The data flow becomes:

1. Parse the server-provided schema and create the editor's reactive state.
2. Unwrap the reactive schema to its raw object when the normalized computed value runs.
3. Clone and normalize that raw JSON-compatible object.
4. Validate and serialize the normalized result as before.

No API, persisted schema, or user-visible layout changes are required.

## Error Handling

The fix prevents the known unsupported proxy input from reaching `structuredClone`. Existing validation continues to report malformed project fields. Other genuinely non-cloneable schema values remain errors because schemas are required to be JSON-compatible.

## Testing

Add a component regression test that mounts `ConfigEditor` with a valid schema. Assert that mounting does not throw and that the fallback textarea contains the normalized serialized schema. This exercises the reactive `ref` and computed path that the existing helper-only tests miss.

Run the focused Vitest file and `make lint-js`.
