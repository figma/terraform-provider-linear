package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTeamWorkflowResourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTeamWorkflowResourceConfigDefault("TFAC"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("linear_team_workflow.test", "id", "ab69b823-f187-444d-8848-782045b86656"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "key", "TFAC"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "branch.pattern"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "branch.is_regex"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "draft"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "start"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "review"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "mergeable"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "merge"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "linear_team_workflow.test",
				ImportState:       true,
				ImportStateId:     "TFAC",
				ImportStateVerify: true,
			},
			// Update with null values
			{
				Config: testAccTeamWorkflowResourceConfigDefault("TFAC"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("linear_team_workflow.test", "id", "ab69b823-f187-444d-8848-782045b86656"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "key", "TFAC"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "branch.pattern"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "branch.is_regex"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "draft"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "start"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "review"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "mergeable"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "merge"),
				),
			},
			// Update and Read testing
			{
				Config: testAccTeamWorkflowResourceConfigNonDefault("TFAC"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("linear_team_workflow.test", "id", "ab69b823-f187-444d-8848-782045b86656"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "key", "TFAC"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "branch.pattern"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "branch.is_regex"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "draft", "5c5382ee-9b07-425a-8a95-138227f2c9de"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "start", "4823a72a-3850-4be1-bcfd-93ea2f0c45ff"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "review", "4823a72a-3850-4be1-bcfd-93ea2f0c45ff"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "mergeable", "df4b3f19-1285-46a4-899d-7f3d51357ec4"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "merge", "57a06c56-4200-42e3-9d01-06965cf1ec90"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "linear_team_workflow.test",
				ImportState:       true,
				ImportStateId:     "TFAC",
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccTeamWorkflowResourceDefaultUpdateBranch(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTeamWorkflowResourceConfigDefault("TFAC"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("linear_team_workflow.test", "id", "ab69b823-f187-444d-8848-782045b86656"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "key", "TFAC"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "branch.pattern"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "branch.is_regex"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "draft"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "start"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "review"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "mergeable"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "merge"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "linear_team_workflow.test",
				ImportState:       true,
				ImportStateId:     "TFAC",
				ImportStateVerify: true,
			},
			// Update with null values
			{
				Config: testAccTeamWorkflowResourceConfigDefault("TFAC"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("linear_team_workflow.test", "id", "ab69b823-f187-444d-8848-782045b86656"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "key", "TFAC"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "branch.pattern"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "branch.is_regex"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "draft"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "start"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "review"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "mergeable"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "merge"),
				),
			},
			// Update and Read testing
			{
				Config: testAccTeamWorkflowResourceConfigNonDefaultBranch("TFAC"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("linear_team_workflow.test", "id", "ab69b823-f187-444d-8848-782045b86656"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "key", "TFAC"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "branch.pattern", "feature/.*"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "branch.is_regex", "true"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "draft", "5c5382ee-9b07-425a-8a95-138227f2c9de"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "start", "4823a72a-3850-4be1-bcfd-93ea2f0c45ff"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "review", "4823a72a-3850-4be1-bcfd-93ea2f0c45ff"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "mergeable", "df4b3f19-1285-46a4-899d-7f3d51357ec4"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "merge", "57a06c56-4200-42e3-9d01-06965cf1ec90"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "linear_team_workflow.test",
				ImportState:       true,
				ImportStateId:     "TFAC:feature/.*:true",
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccTeamWorkflowResourceNonDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTeamWorkflowResourceConfigNonDefaultBranch("TFAC"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("linear_team_workflow.test", "id", "ab69b823-f187-444d-8848-782045b86656"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "key", "TFAC"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "branch.pattern", "feature/.*"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "branch.is_regex", "true"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "draft", "5c5382ee-9b07-425a-8a95-138227f2c9de"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "start", "4823a72a-3850-4be1-bcfd-93ea2f0c45ff"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "review", "4823a72a-3850-4be1-bcfd-93ea2f0c45ff"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "mergeable", "df4b3f19-1285-46a4-899d-7f3d51357ec4"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "merge", "57a06c56-4200-42e3-9d01-06965cf1ec90"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "linear_team_workflow.test",
				ImportState:       true,
				ImportStateId:     "TFAC:feature/.*:true",
				ImportStateVerify: true,
			},
			// Update with same values
			{
				Config: testAccTeamWorkflowResourceConfigNonDefaultBranch("TFAC"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("linear_team_workflow.test", "id", "ab69b823-f187-444d-8848-782045b86656"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "key", "TFAC"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "branch.pattern", "feature/.*"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "branch.is_regex", "true"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "draft", "5c5382ee-9b07-425a-8a95-138227f2c9de"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "start", "4823a72a-3850-4be1-bcfd-93ea2f0c45ff"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "review", "4823a72a-3850-4be1-bcfd-93ea2f0c45ff"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "mergeable", "df4b3f19-1285-46a4-899d-7f3d51357ec4"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "merge", "57a06c56-4200-42e3-9d01-06965cf1ec90"),
				),
			},
			// Update with null values
			{
				Config: testAccTeamWorkflowResourceConfigDefaultBranch("TFAC"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("linear_team_workflow.test", "id", "ab69b823-f187-444d-8848-782045b86656"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "key", "TFAC"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "branch.pattern", "feature/.*"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "branch.is_regex", "true"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "draft"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "start"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "review"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "mergeable"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "merge"),
				),
			},
			// ImportState testing
			// TODO: Uncomment when retrieving GitAutomationTargetBranch is supported
			// {
			// 	ResourceName:      "linear_team_workflow.test",
			// 	ImportState:       true,
			// 	ImportStateId:     "TFAC:feature/.*:true",
			// 	ImportStateVerify: true,
			// },
			// Delete testing automatically occurs in TestCase
		},
	})
}
func TestAccTeamWorkflowResourceNonDefaultUpdateBranch(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTeamWorkflowResourceConfigNonDefaultBranch("TFAC"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("linear_team_workflow.test", "id", "ab69b823-f187-444d-8848-782045b86656"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "key", "TFAC"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "branch.pattern", "feature/.*"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "branch.is_regex", "true"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "draft", "5c5382ee-9b07-425a-8a95-138227f2c9de"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "start", "4823a72a-3850-4be1-bcfd-93ea2f0c45ff"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "review", "4823a72a-3850-4be1-bcfd-93ea2f0c45ff"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "mergeable", "df4b3f19-1285-46a4-899d-7f3d51357ec4"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "merge", "57a06c56-4200-42e3-9d01-06965cf1ec90"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "linear_team_workflow.test",
				ImportState:       true,
				ImportStateId:     "TFAC:feature/.*:true",
				ImportStateVerify: true,
			},
			// Update with same values
			{
				Config: testAccTeamWorkflowResourceConfigNonDefaultBranch("TFAC"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("linear_team_workflow.test", "id", "ab69b823-f187-444d-8848-782045b86656"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "key", "TFAC"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "branch.pattern", "feature/.*"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "branch.is_regex", "true"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "draft", "5c5382ee-9b07-425a-8a95-138227f2c9de"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "start", "4823a72a-3850-4be1-bcfd-93ea2f0c45ff"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "review", "4823a72a-3850-4be1-bcfd-93ea2f0c45ff"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "mergeable", "df4b3f19-1285-46a4-899d-7f3d51357ec4"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "merge", "57a06c56-4200-42e3-9d01-06965cf1ec90"),
				),
			},
			// Update with null values
			{
				Config: testAccTeamWorkflowResourceConfigDefault("TFAC"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("linear_team_workflow.test", "id", "ab69b823-f187-444d-8848-782045b86656"),
					resource.TestCheckResourceAttr("linear_team_workflow.test", "key", "TFAC"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "branch.pattern"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "branch.is_regex"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "draft"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "start"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "review"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "mergeable"),
					resource.TestCheckNoResourceAttr("linear_team_workflow.test", "merge"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "linear_team_workflow.test",
				ImportState:       true,
				ImportStateId:     "TFAC",
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccTeamWorkflowResourceConfigDefault(key string) string {
	return fmt.Sprintf(`
resource "linear_team_workflow" "test" {
  key = "%s"
}
`, key)
}

func testAccTeamWorkflowResourceConfigDefaultBranch(key string) string {
	return fmt.Sprintf(`
resource "linear_team_workflow" "test" {
  key = "%s"

  branch = {
    pattern = "feature/.*"
    is_regex = true
  }
}
`, key)
}

func testAccTeamWorkflowResourceConfigNonDefault(key string) string {
	return fmt.Sprintf(`
resource "linear_team_workflow" "test" {
  key = "%s"

  draft = "5c5382ee-9b07-425a-8a95-138227f2c9de"
  start = "4823a72a-3850-4be1-bcfd-93ea2f0c45ff"
  review = "4823a72a-3850-4be1-bcfd-93ea2f0c45ff"
  mergeable = "df4b3f19-1285-46a4-899d-7f3d51357ec4"
  merge = "57a06c56-4200-42e3-9d01-06965cf1ec90"
}
`, key)
}

func testAccTeamWorkflowResourceConfigNonDefaultBranch(key string) string {
	return fmt.Sprintf(`
resource "linear_team_workflow" "test" {
  key = "%s"

  branch = {
    pattern = "feature/.*"
    is_regex = true
  }

  draft = "5c5382ee-9b07-425a-8a95-138227f2c9de"
  start = "4823a72a-3850-4be1-bcfd-93ea2f0c45ff"
  review = "4823a72a-3850-4be1-bcfd-93ea2f0c45ff"
  mergeable = "df4b3f19-1285-46a4-899d-7f3d51357ec4"
  merge = "57a06c56-4200-42e3-9d01-06965cf1ec90"
}
`, key)
}
