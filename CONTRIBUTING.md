# Contributing

Thanks for helping improve the community Google Ads provider.

## Before opening an issue

- Search existing issues and the resource backlog.
- Confirm the behavior against a supported Google Ads API version.
- Remove credentials, customer IDs, resource names, request IDs, URLs, ad copy, and generated account exports from logs and examples.
- For security-sensitive reports, follow `SECURITY.md` instead of opening a public issue.

## Development setup

Requirements:

- Go version declared in `go.mod`;
- Terraform 1.5 or newer;
- Google Ads credentials only for opt-in acceptance tests.

Common commands:

```sh
make fmt
make lint
make test
make docs
git diff --exit-code docs/
```

Unit tests use in-process fake gRPC services and do not require credentials. Acceptance tests mutate real Google Ads resources and are intentionally separate:

```sh
TF_ACC=1 make testacc
```

Run acceptance tests only against a dedicated test customer with billing disabled or another environment where mutations are explicitly safe. Never use a contributor's client account as a test fixture.

## Pull requests

- Keep each pull request focused on one resource, bug, or compatibility change.
- Add unit tests for create/read/update/delete behavior and field masks.
- Add import and no-op-plan coverage when changing resource state behavior.
- Regenerate provider docs with `make docs` after schema changes.
- Call out destructive behavior, replacement semantics, API-version assumptions, and migration steps in the description.
- Do not introduce real account identifiers or credentials, even if they are not authentication secrets.

New resources should follow the existing layering: Google Ads client methods in `internal/googleads`, Terraform schema and lifecycle behavior in `internal/provider`, generated examples/docs, and focused tests.

## Compatibility

The provider is pre-1.0. We still avoid unnecessary schema breaks, but a minor release may contain migration-requiring changes. Deprecate fields before removal when practical and document state migration or import steps.

