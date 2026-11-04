package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccWorkspaceDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccWorkspaceDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.linear_workspace.test", "id", "1342e121-6bad-4d37-a414-8a1dc788fc5b"),
					resource.TestCheckResourceAttr("data.linear_workspace.test", "name", "Figma Sandbox"),
					resource.TestCheckResourceAttr("data.linear_workspace.test", "url_key", "figma-sandbox"),
				),
			},
		},
	})
}

const testAccWorkspaceDataSourceConfig = `
data "linear_workspace" "test" {}
`
