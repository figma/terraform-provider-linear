package provider

import (
	"context"
	"fmt"
	"sort"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewInitiativeLabelDataSource() datasource.DataSource  { return &InitiativeLabelDataSource{} }
func NewInitiativeLabelsDataSource() datasource.DataSource { return &InitiativeLabelsDataSource{} }

type InitiativeLabelDataSource struct{ client *graphql.Client }
type InitiativeLabelsDataSource struct{ InitiativeLabelDataSource }
type InitiativeLabelsDataSourceModel struct {
	Id              types.String                 `tfsdk:"id"`
	Name            types.String                 `tfsdk:"name"`
	ParentId        types.String                 `tfsdk:"parent_id"`
	IsGroup         types.Bool                   `tfsdk:"is_group"`
	IncludeArchived types.Bool                   `tfsdk:"include_archived"`
	Labels          []InitiativeLabelLookupModel `tfsdk:"labels"`
}

func initiativeLabelLookupAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id":          schema.StringAttribute{Computed: true, MarkdownDescription: "Label UUID."},
		"name":        schema.StringAttribute{Computed: true, MarkdownDescription: "Label name."},
		"description": schema.StringAttribute{Computed: true, MarkdownDescription: "Label description."},
		"color":       schema.StringAttribute{Computed: true, MarkdownDescription: "Hex color."},
		"is_group":    schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether this label is a group."},
		"group_type":  schema.StringAttribute{Computed: true, MarkdownDescription: "Group selection behavior, when applicable."},
		"parent_id":   schema.StringAttribute{Computed: true, MarkdownDescription: "Parent label group UUID, or null for a root label."},
	}
}

func (d *InitiativeLabelDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_initiative_label"
}
func (d *InitiativeLabelsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_initiative_labels"
}

func (d *InitiativeLabelDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	a := initiativeLabelLookupAttributes()
	a["id"] = schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "Exact label UUID. Specify exactly one of id or name.", Validators: []validator.String{stringvalidator.ExactlyOneOf(path.MatchRoot("name")), stringvalidator.RegexMatches(uuidRegex(), "must be a UUID")}}
	a["name"] = schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "Exact label name. Ambiguous names produce an error."}
	a["parent_id"] = schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "Optional parent UUID to disambiguate a name.", Validators: []validator.String{stringvalidator.RegexMatches(uuidRegex(), "must be a UUID")}}
	resp.Schema = schema.Schema{MarkdownDescription: "Look up one initiative label or group by UUID or exact name. Name lookups search active labels; use UUID to retrieve an archived label.", Attributes: a}
}

func (d *InitiativeLabelsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "List initiative labels and groups with full cursor pagination. Filters match exactly and combine with AND.", Attributes: map[string]schema.Attribute{
		"id":               schema.StringAttribute{Computed: true, MarkdownDescription: "Identifier of this collection."},
		"name":             schema.StringAttribute{Optional: true, MarkdownDescription: "Exact name filter."},
		"parent_id":        schema.StringAttribute{Optional: true, MarkdownDescription: "Parent group UUID filter.", Validators: []validator.String{stringvalidator.RegexMatches(uuidRegex(), "must be a UUID")}},
		"is_group":         schema.BoolAttribute{Optional: true, MarkdownDescription: "Filter for groups or ordinary labels."},
		"include_archived": schema.BoolAttribute{Optional: true, MarkdownDescription: "Include archived labels. Defaults to false."},
		"labels":           schema.ListNestedAttribute{Computed: true, MarkdownDescription: "Matching labels, sorted by UUID.", NestedObject: schema.NestedAttributeObject{Attributes: initiativeLabelLookupAttributes()}},
	}}
}

func (d *InitiativeLabelDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*graphql.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *graphql.Client, got %T.", req.ProviderData))
		return
	}
	d.client = client
}

func (d *InitiativeLabelDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data InitiativeLabelLookupModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var label InitiativeLabelLookup
	if !data.Id.IsNull() {
		result, err := lookupInitiativeLabel(ctx, *d.client, data.Id.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read initiative label: %s", err))
			return
		}
		label = result.InitiativeLabel.InitiativeLabelLookup
		if !data.ParentId.IsNull() && (label.Parent == nil || label.Parent.Id != data.ParentId.ValueString()) {
			resp.Diagnostics.AddError("Label not found", "The label does not have the requested parent.")
			return
		}
	} else {
		labels, err := allInitiativeLabels(ctx, *d.client, false)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list initiative labels: %s", err))
			return
		}
		matches := filterInitiativeLabels(labels, data.Name, data.ParentId, types.BoolNull())
		if len(matches) != 1 {
			resp.Diagnostics.AddError("Label lookup failed", fmt.Sprintf("Expected exactly one initiative label, found %d. Use id or parent_id to disambiguate.", len(matches)))
			return
		}
		label = matches[0]
	}
	setInitiativeLabelLookupState(&data, label)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *InitiativeLabelsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data InitiativeLabelsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	labels, err := allInitiativeLabels(ctx, *d.client, data.IncludeArchived.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list initiative labels: %s", err))
		return
	}
	data.Id = types.StringValue("initiative_labels")
	data.Labels = []InitiativeLabelLookupModel{}
	for _, label := range filterInitiativeLabels(labels, data.Name, data.ParentId, data.IsGroup) {
		var model InitiativeLabelLookupModel
		setInitiativeLabelLookupState(&model, label)
		data.Labels = append(data.Labels, model)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func allInitiativeLabels(ctx context.Context, client graphql.Client, includeArchived bool) ([]InitiativeLabelLookup, error) {
	labels := []InitiativeLabelLookup{}
	var after *string
	for {
		response, err := listInitiativeLabels(ctx, client, after, includeArchived)
		if err != nil {
			return nil, err
		}
		for _, node := range response.InitiativeLabels.Nodes {
			labels = append(labels, node.InitiativeLabelLookup)
		}
		page := response.InitiativeLabels.PageInfo
		if !page.HasNextPage {
			break
		}
		if page.EndCursor == nil || *page.EndCursor == "" || (after != nil && *after == *page.EndCursor) {
			return nil, fmt.Errorf("Linear returned an invalid pagination cursor")
		}
		after = page.EndCursor
	}
	sort.Slice(labels, func(i, j int) bool { return labels[i].Id < labels[j].Id })
	return labels, nil
}

func filterInitiativeLabels(labels []InitiativeLabelLookup, name, parent types.String, isGroup types.Bool) []InitiativeLabelLookup {
	result := []InitiativeLabelLookup{}
	for _, label := range labels {
		if !name.IsNull() && label.Name != name.ValueString() {
			continue
		}
		if !parent.IsNull() && (label.Parent == nil || label.Parent.Id != parent.ValueString()) {
			continue
		}
		if !isGroup.IsNull() && label.IsGroup != isGroup.ValueBool() {
			continue
		}
		result = append(result, label)
	}
	return result
}

type InitiativeLabelLookupModel struct {
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Color       types.String `tfsdk:"color"`
	IsGroup     types.Bool   `tfsdk:"is_group"`
	GroupType   types.String `tfsdk:"group_type"`
	ParentId    types.String `tfsdk:"parent_id"`
}

func setInitiativeLabelLookupState(data *InitiativeLabelLookupModel, initiativeLabel InitiativeLabelLookup) {
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
