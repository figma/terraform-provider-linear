# Webhook management and signing secrets

`linear_webhook` manages one subscription. `linear_webhook` and
`linear_webhooks` data sources return subscription metadata without signing
secrets. The provider never selects `Webhook.secret`, including during import.

## Secret lifecycle

Provide `secret_wo` through an ephemeral, sensitive variable populated by your
secret manager, and set `secret_wo_version` to a positive integer. Use the same
secret in the receiver, which must verify Linear's HMAC-SHA256 signature over the
raw request body. See [Linear's webhook documentation](https://linear.app/developers/webhooks).

The provider reads the secret from configuration, never from the plan or state.
Resource state contains only `secret_wo_version`. Supplying the secret through
an ephemeral variable also excludes the variable value from saved plans; a
regular sensitive variable can still persist in plan artifacts. Do not hardcode
secrets in configuration. Supply ephemeral inputs during both plan and apply.

Change `secret_wo_version` whenever rotating the secret. Changing only
`secret_wo` cannot produce a plan difference. Unrelated updates do not resend the
secret. Coordinate rotation with the receiver: Linear does not promise an overlap
window for old and new signatures. Removing both arguments leaves the remote
secret untouched. Refresh cannot detect external rotation.

Omit both arguments to leave the signing secret unmanaged. On creation Linear
generates it; obtain that value through Linear's API settings and configure the
receiver separately. Import does not recover a secret or rotation version.
Configuring both arguments after import sets the supplied secret on the next apply.

## Compatibility and rollout

Keep this PR draft until Figma's OpenTofu upgrade lands, then rebase and repeat
live acceptance checks. The Linear Spacelift stack currently uses OpenTofu 1.9.1;
Foundation has stated a target of 1.13.0 without a committed rollout date.

Write-only signing secrets require **OpenTofu 1.11+** or **Terraform 1.11.1+**
for the recommended sensitive ephemeral variable. Terraform 1.11.0 introduced
write-only arguments but has a sensitive/ephemeral serialization bug fixed in
[1.11.1](https://github.com/hashicorp/terraform/pull/36619).
Older clients can still use the provider with `secret_wo` omitted, including
webhooks whose signing secrets are managed outside Terraform.

Plugin Framework 1.16.1 supplies write-only support, with plugin-testing 1.13.3
and its matching protocol/SDK dependencies. CI acceptance tests use Terraform
1.11.1. Framework 1.14.0 was the initial implementation; 1.14.1 fixed perpetual
diffs involving computed attributes.

## Practices adopted from established providers

- The [GitHub provider](https://github.com/integrations/terraform-provider-github/blob/main/github/resource_github_repository_webhook.go)
  avoids replacing configured webhook secrets with masked API responses. This
  provider never queries the remote signing secret.
- The [GitLab provider](https://github.com/gitlabhq/terraform-provider-gitlab/blob/main/docs/resources/project_hook.md)
  documents that imported subscriptions cannot recover their secrets. This
  provider likewise excludes signing secrets from lookups and import.
- [HashiCorp's write-only guidance](https://developer.hashicorp.com/terraform/plugin/framework/resources/write-only-arguments)
  recommends a companion version attribute to trigger rotation. Sensitive alone
  provides output redaction, so this resource uses write-only configuration.
  Raw API errors are omitted because they can echo credentials.

Do not embed credentials in receiver URLs; URLs remain ordinary metadata and
are returned by lookups. An ephemeral source cannot remove a secret already
stored by another resource or data source upstream.

## Scope, updates, and import

Specify exactly one team UUID or `all_public_teams = true`. Scope changes replace
the subscription because `WebhookUpdateInput` cannot change scope. The
all-public scope covers future public teams too. Delivery can be paused using
`enabled = false`. Event resource names form a Terraform set, so reordering does
not cause updates. Linear validates supported event types and their permissions;
the provider does not freeze an event-name allowlist that can become stale.

Creation requires workspace API-management permission and, for team-specific
subscriptions, team-management permission. Certain events have additional
requirements. Imported OAuth application webhooks without their own URL cannot
be managed by this resource; manage those through their application instead.

Lookups paginate all subscriptions and sort by UUID. Collection filters match
exactly. Lookups can display OAuth subscription metadata, including a null URL.

## Verification

Local tests exercise configuration-only secret reads, secret-free state and import,
version-triggered rotation, omission on unrelated changes, credential-safe diagnostics, scope/URL
validation, and paginated metadata lookups. Live acceptance tests create a
**disabled** subscription, update its secret and events, verify both lookups,
import without recovering the secret or version, and check a clean plan. They require the
existing CI Linear credential and never enable event delivery.
