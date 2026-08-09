# Organization Project Config Clone Fix

## Problem

`ConfigEditor` stores its schema in a Vue `ref`, which exposes the schema as a reactive `Proxy`. The `normalized` computed value passes that proxy to `normalizeSchema`, whose `cloneJSON` helper uses `structuredClone`. Browsers cannot structured-clone proxies, so opening the configuration editor raises `DataCloneError` and interrupts rendering.

## Design

Keep `cloneJSON` and `normalizeSchema` independent of Vue. Change `cloneJSON` to use a JSON serialization round trip, which accepts Vue proxies and reads their nested properties through the proxy. All current callers operate on data received from or written to JSON, so JSON cloning matches their data contract.

Unwrapping `schema.value` with Vue's `toRaw` was rejected during implementation testing. Although it prevents `DataCloneError`, it also bypasses deep reactive dependency tracking, leaving the serialized schema stale after nested editor changes.

The data flow becomes:

1. Parse the server-provided schema and create the editor's reactive state.
2. Serialize the reactive schema when the normalized computed value runs, tracking its nested properties.
3. Parse the resulting plain JSON value and normalize it.
4. Validate and serialize the normalized result as before.

No API, persisted schema, or user-visible layout changes are required.

## Error Handling

The fix removes the unsupported proxy input from the clone path. Existing validation continues to report malformed project fields. Schemas remain limited to JSON-compatible values by their existing server and form data contract.

## Testing

Add a component regression test that mounts `ConfigEditor` with a valid schema. Assert that mounting does not throw, the fallback textarea contains the normalized serialized schema, and adding a field updates that serialized value. This exercises both the reactive `ref` and nested update paths that the existing helper-only tests miss.

Run the focused Vitest file and `make lint-js`.
