package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccWebhookResource(t *testing.T) {
	label := fmt.Sprintf("tf-acc-webhook-%d", time.Now().UnixNano())
	config := func(secret, resources string) string {
		return fmt.Sprintf(`
resource "linear_webhook" "test" {
  label = %q
  url = "https://example.com/terraform-provider-linear"
  enabled = false
  all_public_teams = true
  resource_types = %s
  secret = %q
}
data "linear_webhook" "test" {
  id = linear_webhook.test.id
}
data "linear_webhooks" "test" {
  label = linear_webhook.test.label
  enabled = false
}
`, label, resources, secret)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config("tf-acc-secret-original", `["Issue"]`), Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestMatchResourceAttr("linear_webhook.test", "id", uuidRegex()),
				resource.TestCheckResourceAttr("linear_webhook.test", "enabled", "false"),
				resource.TestCheckResourceAttrPair("linear_webhook.test", "id", "data.linear_webhook.test", "id"),
				resource.TestCheckResourceAttr("data.linear_webhooks.test", "webhooks.#", "1"),
			)},
			{Config: config("tf-acc-secret-rotated", `["Issue", "Comment"]`), Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("linear_webhook.test", "secret", "tf-acc-secret-rotated"),
				resource.TestCheckResourceAttr("linear_webhook.test", "resource_types.#", "2"),
			)},
			{ResourceName: "linear_webhook.test", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"secret"}},
			{Config: config("tf-acc-secret-rotated", `["Comment", "Issue"]`), PlanOnly: true},
		},
	})
}
