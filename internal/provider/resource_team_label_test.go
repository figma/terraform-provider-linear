package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTeamLabelResourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTeamLabelResourceConfigDefault("TFAC Tech Debt"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("linear_team_label.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team_label.test", "name", "TFAC Tech Debt"),
					resource.TestCheckNoResourceAttr("linear_team_label.test", "description"),
					resource.TestMatchResourceAttr("linear_team_label.test", "color", colorRegex()),
					resource.TestCheckResourceAttr("linear_team_label.test", "is_group", "false"),
					resource.TestCheckNoResourceAttr("linear_team_label.test", "parent_id"),
					resource.TestCheckResourceAttr("linear_team_label.test", "team_id", "ab69b823-f187-444d-8848-782045b86656"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "linear_team_label.test",
				ImportState:       true,
				ImportStateId:     "TFAC Tech Debt:TFAC",
				ImportStateVerify: true,
			},
			// Update with null values
			{
				Config: testAccTeamLabelResourceConfigDefault("TFAC Tech Debt"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("linear_team_label.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team_label.test", "name", "TFAC Tech Debt"),
					resource.TestCheckNoResourceAttr("linear_team_label.test", "description"),
					resource.TestMatchResourceAttr("linear_team_label.test", "color", colorRegex()),
					resource.TestCheckResourceAttr("linear_team_label.test", "is_group", "false"),
					resource.TestCheckNoResourceAttr("linear_team_label.test", "parent_id"),
					resource.TestCheckResourceAttr("linear_team_label.test", "team_id", "ab69b823-f187-444d-8848-782045b86656"),
				),
			},
			// Update and Read testing
			{
				Config: testAccTeamLabelResourceConfigNonDefault("TFAC Easy Tech Debt"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("linear_team_label.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team_label.test", "name", "TFAC Easy Tech Debt"),
					resource.TestCheckResourceAttr("linear_team_label.test", "description", "lots of it"),
					resource.TestCheckResourceAttr("linear_team_label.test", "color", "#00ff00"),
					resource.TestCheckResourceAttr("linear_team_label.test", "is_group", "false"),
					resource.TestCheckResourceAttr("linear_team_label.test", "parent_id", "86f1e64c-313c-4f08-92cc-35e8e10b20a1"),
					resource.TestCheckResourceAttr("linear_team_label.test", "team_id", "ab69b823-f187-444d-8848-782045b86656"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "linear_team_label.test",
				ImportState:       true,
				ImportStateId:     "TFAC Easy Tech Debt:TFAC",
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccTeamLabelResourceDefaultGroup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTeamLabelResourceConfigDefault("TFAC Tech Debt"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("linear_team_label.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team_label.test", "name", "TFAC Tech Debt"),
					resource.TestCheckNoResourceAttr("linear_team_label.test", "description"),
					resource.TestMatchResourceAttr("linear_team_label.test", "color", colorRegex()),
					resource.TestCheckResourceAttr("linear_team_label.test", "is_group", "false"),
					resource.TestCheckNoResourceAttr("linear_team_label.test", "parent_id"),
					resource.TestCheckResourceAttr("linear_team_label.test", "team_id", "ab69b823-f187-444d-8848-782045b86656"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "linear_team_label.test",
				ImportState:       true,
				ImportStateId:     "TFAC Tech Debt:TFAC",
				ImportStateVerify: true,
			},
			// Update with null values
			{
				Config: testAccTeamLabelResourceConfigDefault("TFAC Tech Debt"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("linear_team_label.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team_label.test", "name", "TFAC Tech Debt"),
					resource.TestCheckNoResourceAttr("linear_team_label.test", "description"),
					resource.TestMatchResourceAttr("linear_team_label.test", "color", colorRegex()),
					resource.TestCheckResourceAttr("linear_team_label.test", "is_group", "false"),
					resource.TestCheckNoResourceAttr("linear_team_label.test", "parent_id"),
					resource.TestCheckResourceAttr("linear_team_label.test", "team_id", "ab69b823-f187-444d-8848-782045b86656"),
				),
			},
			// Update and Read testing
			{
				Config: testAccTeamLabelResourceConfigGroup("TFAC Easy Tech Debt"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("linear_team_label.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team_label.test", "name", "TFAC Easy Tech Debt"),
					resource.TestCheckNoResourceAttr("linear_team_label.test", "description"),
					resource.TestMatchResourceAttr("linear_team_label.test", "color", colorRegex()),
					resource.TestCheckResourceAttr("linear_team_label.test", "is_group", "true"),
					resource.TestCheckNoResourceAttr("linear_team_label.test", "parent_id"),
					resource.TestCheckResourceAttr("linear_team_label.test", "team_id", "ab69b823-f187-444d-8848-782045b86656"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "linear_team_label.test",
				ImportState:       true,
				ImportStateId:     "TFAC Easy Tech Debt:TFAC",
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccTeamLabelResourceNonDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTeamLabelResourceConfigNonDefault("TFAC Needs Design"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("linear_team_label.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team_label.test", "name", "TFAC Needs Design"),
					resource.TestCheckResourceAttr("linear_team_label.test", "description", "lots of it"),
					resource.TestCheckResourceAttr("linear_team_label.test", "color", "#00ff00"),
					resource.TestCheckResourceAttr("linear_team_label.test", "is_group", "false"),
					resource.TestCheckResourceAttr("linear_team_label.test", "parent_id", "86f1e64c-313c-4f08-92cc-35e8e10b20a1"),
					resource.TestCheckResourceAttr("linear_team_label.test", "team_id", "ab69b823-f187-444d-8848-782045b86656"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "linear_team_label.test",
				ImportState:       true,
				ImportStateId:     "TFAC Needs Design:TFAC",
				ImportStateVerify: true,
			},
			// Update with same values
			{
				Config: testAccTeamLabelResourceConfigNonDefault("TFAC Needs Design"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("linear_team_label.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team_label.test", "name", "TFAC Needs Design"),
					resource.TestCheckResourceAttr("linear_team_label.test", "description", "lots of it"),
					resource.TestCheckResourceAttr("linear_team_label.test", "color", "#00ff00"),
					resource.TestCheckResourceAttr("linear_team_label.test", "is_group", "false"),
					resource.TestCheckResourceAttr("linear_team_label.test", "parent_id", "86f1e64c-313c-4f08-92cc-35e8e10b20a1"),
					resource.TestCheckResourceAttr("linear_team_label.test", "team_id", "ab69b823-f187-444d-8848-782045b86656"),
				),
			},
			// Update with null values
			{
				Config: testAccTeamLabelResourceConfigDefault("TFAC Tech Debt"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("linear_team_label.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team_label.test", "name", "TFAC Tech Debt"),
					resource.TestCheckNoResourceAttr("linear_team_label.test", "description"),
					resource.TestMatchResourceAttr("linear_team_label.test", "color", colorRegex()),
					resource.TestCheckResourceAttr("linear_team_label.test", "is_group", "false"),
					resource.TestCheckNoResourceAttr("linear_team_label.test", "parent_id"),
					resource.TestCheckResourceAttr("linear_team_label.test", "team_id", "ab69b823-f187-444d-8848-782045b86656"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "linear_team_label.test",
				ImportState:       true,
				ImportStateId:     "TFAC Tech Debt:TFAC",
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccTeamLabelResourceNonDefaultGroup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTeamLabelResourceConfigGroup("TFAC Needs Design"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("linear_team_label.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team_label.test", "name", "TFAC Needs Design"),
					resource.TestCheckNoResourceAttr("linear_team_label.test", "description"),
					resource.TestMatchResourceAttr("linear_team_label.test", "color", colorRegex()),
					resource.TestCheckResourceAttr("linear_team_label.test", "is_group", "true"),
					resource.TestCheckNoResourceAttr("linear_team_label.test", "parent_id"),
					resource.TestCheckResourceAttr("linear_team_label.test", "team_id", "ab69b823-f187-444d-8848-782045b86656"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "linear_team_label.test",
				ImportState:       true,
				ImportStateId:     "TFAC Needs Design:TFAC",
				ImportStateVerify: true,
			},
			// Update with same values
			{
				Config: testAccTeamLabelResourceConfigGroup("TFAC Needs Design"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("linear_team_label.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team_label.test", "name", "TFAC Needs Design"),
					resource.TestCheckNoResourceAttr("linear_team_label.test", "description"),
					resource.TestMatchResourceAttr("linear_team_label.test", "color", colorRegex()),
					resource.TestCheckResourceAttr("linear_team_label.test", "is_group", "true"),
					resource.TestCheckNoResourceAttr("linear_team_label.test", "parent_id"),
					resource.TestCheckResourceAttr("linear_team_label.test", "team_id", "ab69b823-f187-444d-8848-782045b86656"),
				),
			},
			// Update with null values
			{
				Config: testAccTeamLabelResourceConfigDefault("TFAC Tech Debt"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("linear_team_label.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team_label.test", "name", "TFAC Tech Debt"),
					resource.TestCheckNoResourceAttr("linear_team_label.test", "description"),
					resource.TestMatchResourceAttr("linear_team_label.test", "color", colorRegex()),
					resource.TestCheckResourceAttr("linear_team_label.test", "is_group", "false"),
					resource.TestCheckNoResourceAttr("linear_team_label.test", "parent_id"),
					resource.TestCheckResourceAttr("linear_team_label.test", "team_id", "ab69b823-f187-444d-8848-782045b86656"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "linear_team_label.test",
				ImportState:       true,
				ImportStateId:     "TFAC Tech Debt:TFAC",
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccTeamLabelResourceConfigDefault(name string) string {
	return fmt.Sprintf(`
resource "linear_team_label" "test" {
  name = "%s"
  team_id = "ab69b823-f187-444d-8848-782045b86656"
}
`, name)
}

func testAccTeamLabelResourceConfigNonDefault(name string) string {
	return fmt.Sprintf(`
resource "linear_team_label" "test" {
  name = "%s"
  description = "lots of it"
  color = "#00ff00"
  parent_id = "86f1e64c-313c-4f08-92cc-35e8e10b20a1"
  team_id = "ab69b823-f187-444d-8848-782045b86656"
}
`, name)
}

func testAccTeamLabelResourceConfigGroup(name string) string {
	return fmt.Sprintf(`
resource "linear_team_label" "test" {
  name = "%s"
  is_group = true
  team_id = "ab69b823-f187-444d-8848-782045b86656"
}
`, name)
}
