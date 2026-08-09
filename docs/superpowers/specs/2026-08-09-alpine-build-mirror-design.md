# Alpine Build Mirror Design

## Problem

Production image builds spend most of their time downloading Alpine packages from the default repository. A captured deployment log showed the runtime package installation taking about 11 minutes and the build dependency installation exceeding 56 minutes before completion.

The final image also declares commit-specific labels before installing runtime packages. Because those labels change for every release, the runtime package layer cannot be reused reliably between releases.

## Decision

Use `https://mirrors.aliyun.com/alpine` as the default Alpine package mirror while allowing operators to override it through `ALPINE_MIRROR`.

Apply the setting consistently to every stage that runs `apk add` in both `Dockerfile` and `Dockerfile.rootless`. Pass the value from `ops/compose/compose.yaml`, and document it in `ops/compose/.env.example`.

Move release-specific OCI labels in the final image stages after stable operating-system package and user setup layers. This keeps package installation reusable when only the release commit, version, or build date changes.

## Scope

- Modify `Dockerfile` and `Dockerfile.rootless`.
- Add the `ALPINE_MIRROR` build argument to the production Compose configuration.
- Add the default value to the example production environment.
- Add focused static coverage to the existing operations test suite if it does not already verify these properties.
- Do not change backup, rollback, frontend, Go compilation, or deployment switching behavior.

## Behavior

Each Alpine build stage will:

1. Accept an `ALPINE_MIRROR` build argument with the approved default.
2. Replace only the standard Alpine repository base URL in `/etc/apk/repositories`.
3. Run the existing `apk --no-cache add` command unchanged.

Operators may restore the official source or select another mirror by setting `ALPINE_MIRROR` in `ops/compose/.env` before deployment.

## Failure Handling

An unreachable or incomplete mirror will make `apk add` fail normally, leaving the previous production container active under the existing deployment behavior. Operators can change `ALPINE_MIRROR` and rerun the deployment.

## Verification

- Run `ops/tests/check.sh`.
- Render the production Compose configuration with an isolated test environment.
- Verify all `apk add` stages replace the repository before package installation.
- Verify release-specific labels follow stable runtime dependency installation.
- Confirm the working tree contains only the intended files.
