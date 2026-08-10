# Install Pinned pnpm from a Domestic npm Registry

## Context

The production image build installs `pnpm` from the configured Alpine package
mirror. A mirror synchronization gap caused the indexed
`pnpm-11.20.0-r0` package to return HTTP 404, which stopped deployment before
the production container was switched.

The repository already declares `pnpm@11.9.0` in `package.json`. The image
build should install that exact version without depending on the Alpine
repository's `pnpm` package.

## Design

The frontend build stage will continue installing `nodejs` from Alpine, add
the Alpine `npm` package, and stop installing the Alpine `pnpm` package. It
will then install `pnpm@11.9.0` globally through npm.

The npm registry will be controlled by a new `NPM_REGISTRY` Docker build
argument. Its default will be `https://registry.npmmirror.com`, matching the
domestic-mirror strategy already used for Alpine and Go dependencies.

The production Compose file will pass `NPM_REGISTRY` into the image build and
default it to the same domestic registry. `ops/compose/.env.example` will
document the setting so operators can override it without editing source.

## Failure Behavior

If the npm registry is unavailable or does not contain `pnpm@11.9.0`, the
Docker build will fail normally. The deployment script will preserve the
currently running production release, consistent with existing pre-switch
build failure handling.

## Verification

Operations checks will assert that:

- the frontend build installs `nodejs` and `npm` without Alpine's `pnpm`
  package;
- npm installs exactly `pnpm@11.9.0`;
- the npm registry is configurable through the Compose build argument; and
- the example production environment specifies the domestic default.

The existing operations test suite and a Docker build of the frontend stage
will verify the resulting configuration and installation path.
