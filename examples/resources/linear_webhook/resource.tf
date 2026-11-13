variable "webhook_signing_secret" {
  type      = string
  sensitive = true
  ephemeral = true
}

resource "linear_webhook" "security" {
  label             = "Security issue events"
  url               = "https://hooks.example.com/linear"
  all_public_teams  = true
  resource_types    = ["Issue", "Comment"]
  secret_wo         = var.webhook_signing_secret
  secret_wo_version = 1
}
