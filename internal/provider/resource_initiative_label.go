package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &InitiativeLabelResource{}
var _ resource.ResourceWithImportState = &InitiativeLabelResource{}

func NewInitiativeLabelResource() resource.Resource {
	return &InitiativeLabelResource{}
}

type InitiativeLabelResource struct {
	client *graphql.Client
}

type InitiativeLabelResourceModel struct {
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Color       types.String `tfsdk:"color"`
	IsGroup     types.Bool   `tfsdk:"is_group"`
	GroupType   types.String `tfsdk:"group_type"`
	ParentId    types.String `tfsdk:"parent_id"`
}

func (r *InitiativeLabelResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_initiative_label"
}

func (r *InitiativeLabelResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Workspace initiative label or label group. Deleting a group also deletes its children and removes their assignments. Import by UUID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the initiative label.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the initiative label.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthBetween(1, 80),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the initiative label.",
				Optional:            true,
			},
			"color": schema.StringAttribute{
				MarkdownDescription: "Color of the initiative label.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(colorRegex(), "must be a hex color"),
				},
			},
			"is_group": schema.BoolAttribute{
				MarkdownDescription: "Whether the initiative label is a group (parent) label. **Default** `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"group_type": schema.StringAttribute{
				MarkdownDescription: "Group selection behavior: singleSelect or multiSelect. Only valid for groups; Linear defaults new groups to singleSelect.",
				Optional:            true, Computed: true,
				Validators: []validator.String{stringvalidator.OneOf("singleSelect", "multiSelect")},
			},
			"parent_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the parent initiative label group.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an uuid"),
				},
			},
		},
	}
}

func (r *InitiativeLabelResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*graphql.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *graphql.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *InitiativeLabelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *InitiativeLabelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := InitiativeLabelCreateInput{
		Name:        data.Name.ValueString(),
		GroupType:   initiativeLabelGroupType(data.GroupType),
		Description: data.Description.ValueStringPointer(),
		IsGroup:     data.IsGroup.ValueBool(),
		ParentId:    data.ParentId.ValueStringPointer(),
	}
	if !data.Color.IsUnknown() {
		input.Color = data.Color.ValueStringPointer()
	}

	response, err := initiativeLabelCreate(ctx, *r.client, input)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create initiative label, got error: %s", err))
		return
	}
	if !response.InitiativeLabelCreate.Success {
		resp.Diagnostics.AddError("Client Error", "Unable to create initiative label: Linear reported an unsuccessful creation")
		return
	}

	tflog.Trace(ctx, "created a initiative label")
	setInitiativeLabelState(data, response.InitiativeLabelCreate.InitiativeLabel.InitiativeLabel)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *InitiativeLabelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *InitiativeLabelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := getInitiativeLabel(ctx, *r.client, data.Id.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "Entity not found: InitiativeLabel") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read initiative label, got error: %s", err))
		return
	}

	setInitiativeLabelState(data, response.InitiativeLabel.InitiativeLabel)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *InitiativeLabelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *InitiativeLabelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := InitiativeLabelUpdateInput{
		Name:        data.Name.ValueString(),
		GroupType:   initiativeLabelGroupType(data.GroupType),
		Description: data.Description.ValueStringPointer(),
		IsGroup:     data.IsGroup.ValueBool(),
		ParentId:    data.ParentId.ValueStringPointer(),
	}
	if !data.Color.IsUnknown() {
		input.Color = data.Color.ValueStringPointer()
	}

	response, err := initiativeLabelUpdate(ctx, *r.client, input, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update initiative label, got error: %s", err))
		return
	}
	if !response.InitiativeLabelUpdate.Success {
		resp.Diagnostics.AddError("Client Error", "Unable to update initiative label: Linear reported an unsuccessful update")
		return
	}

	tflog.Trace(ctx, "updated a initiative label")
	setInitiativeLabelState(data, response.InitiativeLabelUpdate.InitiativeLabel.InitiativeLabel)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *InitiativeLabelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *InitiativeLabelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := deleteInitiativeLabel(ctx, *r.client, data.Id.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	tflog.Trace(ctx, "deleted a initiative label")
}

func deleteInitiativeLabel(ctx context.Context, client graphql.Client, id string) error {
	response, err := initiativeLabelDelete(ctx, client, id)
	if err != nil {
		if strings.Contains(err.Error(), "Entity not found: InitiativeLabel") {
			return nil
		}
		return fmt.Errorf("unable to delete initiative label: %w", err)
	}
	if !response.InitiativeLabelDelete.Success {
		return fmt.Errorf("unable to delete initiative label: Linear reported an unsuccessful deletion")
	}
	return nil
}

func (r *InitiativeLabelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func setInitiativeLabelState(data *InitiativeLabelResourceModel, initiativeLabel InitiativeLabel) {
	data.Id = types.StringValue(initiativeLabel.Id)
	data.Name = types.StringValue(initiativeLabel.Name)
	data.Description = types.StringPointerValue(initiativeLabel.Description)
	data.Color = types.StringValue(initiativeLabel.Color)
	data.IsGroup = types.BoolValue(initiativeLabel.IsGroup)
	data.GroupType = types.StringNull()
	if initiativeLabel.GroupType != nil {
		data.GroupType = types.StringValue(string(*initiativeLabel.GroupType))
	}
	if initiativeLabel.Parent != nil {
		data.ParentId = types.StringValue(initiativeLabel.Parent.Id)
	} else {
		data.ParentId = types.StringNull()
	}
}

func initiativeLabelGroupType(value types.String) *LabelGroupType {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	result := LabelGroupType(value.ValueString())
	return &result
}

func (r *InitiativeLabelResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data InitiativeLabelResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() || data.IsGroup.IsUnknown() {
		return
	}
	if !data.GroupType.IsNull() && !data.GroupType.IsUnknown() && !data.IsGroup.ValueBool() {
		resp.Diagnostics.AddAttributeError(path.Root("group_type"), "Invalid label group", "group_type requires is_group = true.")
	}
	if data.IsGroup.ValueBool() && !data.ParentId.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("parent_id"), "Nested label group", "Groups cannot be nested inside another group.")
	}
}
