package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// labelGroupTypePlanModifier preserves an omitted group selection only while the label remains a group.
type labelGroupTypePlanModifier struct{}

func (labelGroupTypePlanModifier) Description(context.Context) string {
	return "Preserves the group selection for groups and clears it for regular labels."
}

func (m labelGroupTypePlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (labelGroupTypePlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	var isGroup types.Bool
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("is_group"), &isGroup)...)
	if resp.Diagnostics.HasError() || isGroup.IsUnknown() {
		return
	}

	if !isGroup.ValueBool() && req.ConfigValue.IsNull() {
		resp.PlanValue = types.StringNull()
		return
	}

	if req.PlanValue.IsUnknown() && !req.ConfigValue.IsUnknown() && !req.StateValue.IsNull() {
		resp.PlanValue = req.StateValue
	}
}
