package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccWebhookResource(t *testing.T) {
	t.Setenv("TF_VAR_webhook_secret", "tf-acc-secret-original")
	label := fmt.Sprintf("tf-acc-webhook-%d", time.Now().UnixNano())
	config := func(version int, resources string) string {
		return fmt.Sprintf(`
variable "webhook_secret" {
  type = string
  sensitive = true
  ephemeral = true
}
resource "linear_webhook" "test" {
  label = %q
  url = "https://example.com/terraform-provider-linear"
  enabled = false
  all_public_teams = true
  resource_types = %s
  secret_wo = var.webhook_secret
  secret_wo_version = %d
}
data "linear_webhook" "test" {
  id = linear_webhook.test.id
}
data "linear_webhooks" "test" {
  label = linear_webhook.test.label
  enabled = false
}
`, label, resources, version)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config(1, `["Issue"]`), Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestMatchResourceAttr("linear_webhook.test", "id", uuidRegex()),
				resource.TestCheckResourceAttr("linear_webhook.test", "enabled", "false"),
				resource.TestCheckNoResourceAttr("linear_webhook.test", "secret_wo"),
				resource.TestCheckResourceAttrPair("linear_webhook.test", "id", "data.linear_webhook.test", "id"),
				resource.TestCheckResourceAttr("data.linear_webhooks.test", "webhooks.#", "1"),
			)},
			{PreConfig: func() { t.Setenv("TF_VAR_webhook_secret", "tf-acc-secret-rotated") }, Config: config(2, `["Issue", "Comment"]`), Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckNoResourceAttr("linear_webhook.test", "secret_wo"),
				resource.TestCheckResourceAttr("linear_webhook.test", "secret_wo_version", "2"),
				resource.TestCheckResourceAttr("linear_webhook.test", "resource_types.#", "2"),
			)},
			{ResourceName: "linear_webhook.test", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"secret_wo_version"}},
			{Config: config(2, `["Comment", "Issue"]`), PlanOnly: true},
		},
	})
}
