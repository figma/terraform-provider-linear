package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccInitiativeLabelResource(t *testing.T) {
	name := fmt.Sprintf("tf-acc-theme-%d", time.Now().UnixNano())
	config := func(groupType, description string) string {
		groupTypeConfig := ""
		if groupType != "" {
			groupTypeConfig = fmt.Sprintf("group_type = %q", groupType)
		}
		return fmt.Sprintf(`
resource "linear_initiative_label" "group" {
  name = %q
  is_group = true
  %s
}
resource "linear_initiative_label" "child" {
  name = "Platform"
  description = %s
  parent_id = linear_initiative_label.group.id
}
`, name, groupTypeConfig, description)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config("", `"Platform work"`), Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("linear_initiative_label.group", "group_type", "singleSelect"),
				resource.TestCheckResourceAttrPair("linear_initiative_label.child", "parent_id", "linear_initiative_label.group", "id"),
			)},
			{Config: config("", `"Platform work"`), PlanOnly: true},
			{Config: config("multiSelect", `"Platform work"`), Check: resource.TestCheckResourceAttr("linear_initiative_label.group", "group_type", "multiSelect")},
			{ResourceName: "linear_initiative_label.group", ImportState: true, ImportStateVerify: true},
			{ResourceName: "linear_initiative_label.child", ImportState: true, ImportStateVerify: true},
			{Config: config("singleSelect", "null"), Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("linear_initiative_label.group", "group_type", "singleSelect"),
				resource.TestCheckNoResourceAttr("linear_initiative_label.child", "description"),
			)},
			{Config: config("singleSelect", "null"), PlanOnly: true},
		},
	})
}
