---
page_title: "Public release checklist"
subcategory: "Maintainers"
description: |-
  Checklist for publishing a sanitized, signed public release without exposing private account data or changing previously released artifacts.
---

# Public release checklist

Publishing a repository is irreversible in practice: clones and source archives may outlive a later visibility change. Complete this checklist before changing repository visibility or publishing to the Terraform Registry.

## Source and history

- Publish from a clean snapshot or a deliberately sanitized history. Do not expose the existing private history without reviewing commit contents and author metadata.
- Search the full history, tags, release source archives, fixtures, examples, and documentation for credentials, real customer IDs, resource names, URLs, emails, and generated account exports.
- Confirm `.env`, `*.tfstate*`, `terraform.tfvars`, service-account keys, OAuth caches, and `generated/` are ignored and absent from every commit.
- Do not move old private tags to new content or replace an existing release. Start with a new SemVer version so previously installed checksums remain valid.

## Project metadata

- Confirm GitHub recognizes `LICENSE` as MPL-2.0.
- Enable private vulnerability reporting and repository security alerts.
- Review `SECURITY.md`, `CONTRIBUTING.md`, issue templates, CODEOWNERS, and the repository description.
- State clearly that this is an independent community provider with no Google affiliation or support.

## Compatibility and verification

- Confirm the protobuf dependency's Google Ads API version is still supported on the official sunset schedule.
- Run `make fmt`, `make lint`, `make test`, `make docs`, and verify generated docs are clean.
- Run a secret scanner against both the candidate tree and any history that will become public.
- Run an import followed by a no-op plan against a dedicated test account.
- Run a create/update/destroy smoke test with `-parallelism=1` against a dedicated account with billing disabled.

## Signing and Registry

- Configure an RSA or DSA GPG signing key in the Terraform Registry and repository secrets.
- Verify the release contains platform archives, the Registry manifest, SHA256SUMS, and a valid detached signature.
- Publish a new signed version; never overwrite it after release.
- Only then connect the public GitHub repository to the Terraform Registry.

