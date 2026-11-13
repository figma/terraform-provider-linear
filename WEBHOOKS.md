# Webhook management and signing secrets

`linear_webhook` manages one subscription. `linear_webhook` and
`linear_webhooks` data sources return subscription metadata without signing
secrets. The provider never selects `Webhook.secret`, including during import.

## Practices adopted from established providers

- The [GitHub provider](https://github.com/integrations/terraform-provider-github/blob/main/github/resource_github_repository_webhook.go)
  preserves the configured signing secret when GitHub returns a masked value.
  This provider likewise keeps the configured secret through refresh and sends
  it on update only when its configured value changes.
- The [GitLab provider](https://github.com/gitlabhq/terraform-provider-gitlab/blob/main/docs/resources/project_hook.md)
  marks webhook tokens and signing tokens sensitive, and documents that imported
  subscriptions cannot recover their secrets. This provider follows that import
  contract and explicitly excludes secrets from lookup schemas.
- [Terraform sensitive attributes](https://developer.hashicorp.com/terraform/language/manage-sensitive-data)
  redact normal CLI output but still persist values in state. This provider
  documents that distinction, accepts secrets through sensitive variables, and
  omits raw API errors from webhook diagnostics because upstream errors can
  echo submitted credentials or credential-bearing URLs.

## Secret lifecycle

Provide `secret` through a sensitive variable populated by your secret manager,
or a separately generated secret. Use the same value in the receiver, which
must verify Linear's HMAC-SHA256 signature over the raw request body. See
[Linear's webhook documentation](https://linear.app/developers/webhooks).

If `secret` is omitted at creation, Linear generates it. Terraform neither
retrieves nor stores that generated value; obtain it through Linear's API
settings and configure the receiver separately. Omitting `secret` on an imported
subscription leaves the existing credential untouched. Setting it explicitly
manages that value on the next apply.

Changing a configured `secret` rotates it in place. Coordinate the receiver
change: Linear does not promise an overlap window for old and new signatures.
Removing the attribute stops managing the secret without clearing or rotating
it. Refresh cannot detect external rotation because it does not retrieve the
remote secret. An unchanged secret is not resent during unrelated updates.

Terraform's newer write-only attributes can avoid persisting an input secret.
The provider currently uses Terraform Plugin Framework 1.2 and supports older
Terraform versions; that framework does not offer write-only attributes. Adding
that capability requires a separately reviewed framework/protocol compatibility
upgrade. This resource does not claim to keep configured secrets out of state.
Restrict access to state and plan artifacts. Do not embed credentials in the
receiver URL; URL configuration is ordinary metadata and is returned by lookups.

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

Local tests exercise secret serialization, refresh/import preservation, explicit
rotation, omission on unrelated changes, credential-safe diagnostics, scope/URL
validation, and paginated metadata lookups. Live acceptance tests create a
**disabled** subscription, update its secret and events, verify both lookups,
import without recovering the secret, and check a clean plan. They require the
existing CI Linear credential and never enable event delivery.
