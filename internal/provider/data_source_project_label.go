package provider

import (
	"context"
	"fmt"
	"sort"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewProjectLabelDataSource() datasource.DataSource  { return &ProjectLabelDataSource{} }
func NewProjectLabelsDataSource() datasource.DataSource { return &ProjectLabelsDataSource{} }

type ProjectLabelDataSource struct{ client *graphql.Client }
type ProjectLabelsDataSource struct{ ProjectLabelDataSource }
type ProjectLabelDataSourceModel struct {
	Id              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	Color           types.String `tfsdk:"color"`
	IsGroup         types.Bool   `tfsdk:"is_group"`
	GroupType       types.String `tfsdk:"group_type"`
	TeamId          types.String `tfsdk:"team_id"`
	InheritedFromId types.String `tfsdk:"inherited_from_id"`
	ParentId        types.String `tfsdk:"parent_id"`
	WorkspaceOnly   types.Bool   `tfsdk:"workspace_only"`
}

type ProjectLabelsDataSourceModel struct {
	WorkspaceOnly   types.Bool                `tfsdk:"workspace_only"`
	Id              types.String              `tfsdk:"id"`
	Name            types.String              `tfsdk:"name"`
	TeamId          types.String              `tfsdk:"team_id"`
	ParentId        types.String              `tfsdk:"parent_id"`
	IsGroup         types.Bool                `tfsdk:"is_group"`
	IncludeArchived types.Bool                `tfsdk:"include_archived"`
	Labels          []ProjectLabelLookupModel `tfsdk:"labels"`
}

func projectLabelLookupAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id":                schema.StringAttribute{Computed: true, MarkdownDescription: "Label UUID."},
		"name":              schema.StringAttribute{Computed: true, MarkdownDescription: "Label name."},
		"team_id":           schema.StringAttribute{Computed: true, MarkdownDescription: "Owning team UUID, or null for workspace labels."},
		"inherited_from_id": schema.StringAttribute{Computed: true, MarkdownDescription: "Source label UUID for an inherited copy."},
		"description":       schema.StringAttribute{Computed: true, MarkdownDescription: "Label description."},
		"color":             schema.StringAttribute{Computed: true, MarkdownDescription: "Hex color."},
		"is_group":          schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether this label is a group."},
		"group_type":        schema.StringAttribute{Computed: true, MarkdownDescription: "Group selection behavior, when applicable."},
		"parent_id":         schema.StringAttribute{Computed: true, MarkdownDescription: "Parent label group UUID, or null for a root label."},
	}
}

func (d *ProjectLabelDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_label"
}
func (d *ProjectLabelsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_labels"
}

func (d *ProjectLabelDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	a := projectLabelLookupAttributes()
	a["id"] = schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "Exact label UUID. Specify exactly one of id or name.", Validators: []validator.String{stringvalidator.ExactlyOneOf(path.MatchRoot("name")), stringvalidator.RegexMatches(uuidRegex(), "must be a UUID")}}
	a["name"] = schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "Exact label name. Ambiguous names produce an error."}
	a["parent_id"] = schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "Optional parent UUID to disambiguate a name.", Validators: []validator.String{stringvalidator.RegexMatches(uuidRegex(), "must be a UUID")}}
	a["workspace_only"] = schema.BoolAttribute{Optional: true, MarkdownDescription: "Restrict results to workspace labels. Cannot be combined with team_id.", Validators: []validator.Bool{boolvalidator.ConflictsWith(path.MatchRoot("team_id"))}}
	a["team_id"] = schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "Optional team UUID to disambiguate a name.", Validators: []validator.String{stringvalidator.RegexMatches(uuidRegex(), "must be a UUID")}}
	resp.Schema = schema.Schema{MarkdownDescription: "Look up one project label or group by UUID or exact name. Name lookups search active labels; use UUID to retrieve an archived label.", Attributes: a}
}

func (d *ProjectLabelsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "List project labels and groups with full cursor pagination. Filters match exactly and combine with AND.", Attributes: map[string]schema.Attribute{
		"id":               schema.StringAttribute{Computed: true, MarkdownDescription: "Identifier of this collection."},
		"workspace_only":   schema.BoolAttribute{Optional: true, MarkdownDescription: "Restrict results to workspace labels. Cannot be combined with team_id.", Validators: []validator.Bool{boolvalidator.ConflictsWith(path.MatchRoot("team_id"))}},
		"team_id":          schema.StringAttribute{Optional: true, MarkdownDescription: "Owning team UUID filter.", Validators: []validator.String{stringvalidator.RegexMatches(uuidRegex(), "must be a UUID")}},
		"name":             schema.StringAttribute{Optional: true, MarkdownDescription: "Exact name filter."},
		"parent_id":        schema.StringAttribute{Optional: true, MarkdownDescription: "Parent group UUID filter.", Validators: []validator.String{stringvalidator.RegexMatches(uuidRegex(), "must be a UUID")}},
		"is_group":         schema.BoolAttribute{Optional: true, MarkdownDescription: "Filter for groups or ordinary labels."},
		"include_archived": schema.BoolAttribute{Optional: true, MarkdownDescription: "Include archived labels. Defaults to false."},
		"labels":           schema.ListNestedAttribute{Computed: true, MarkdownDescription: "Matching labels, sorted by UUID.", NestedObject: schema.NestedAttributeObject{Attributes: projectLabelLookupAttributes()}},
	}}
}

func (d *ProjectLabelDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ProjectLabelDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ProjectLabelDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var label ProjectLabelLookup
	if !data.Id.IsNull() {
		result, err := lookupProjectLabel(ctx, *d.client, data.Id.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project label: %s", err))
			return
		}
		label = result.ProjectLabel.ProjectLabelLookup
		if data.WorkspaceOnly.ValueBool() && label.Team != nil {
			resp.Diagnostics.AddError("Label not found", "The label does not belong to the workspace scope.")
			return
		}
		if !data.TeamId.IsNull() && (label.Team == nil || label.Team.Id != data.TeamId.ValueString()) {
			resp.Diagnostics.AddError("Label not found", "The label does not belong to the requested team.")
			return
		}
		if !data.ParentId.IsNull() && (label.Parent == nil || label.Parent.Id != data.ParentId.ValueString()) {
			resp.Diagnostics.AddError("Label not found", "The label does not have the requested parent.")
			return
		}
	} else {
		labels, err := allProjectLabels(ctx, *d.client, false)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list project labels: %s", err))
			return
		}
		matches := filterProjectLabels(labels, data.Name, data.ParentId, data.TeamId, types.BoolNull(), data.WorkspaceOnly)
		if len(matches) != 1 {
			resp.Diagnostics.AddError("Label lookup failed", fmt.Sprintf("Expected exactly one project label, found %d. Use id, parent_id, team_id, or workspace_only to disambiguate.", len(matches)))
			return
		}
		label = matches[0]
	}
	var result ProjectLabelLookupModel
	setProjectLabelLookupState(&result, label)
	data.Id = result.Id
	data.Name = result.Name
	data.Description = result.Description
	data.Color = result.Color
	data.IsGroup = result.IsGroup
	data.GroupType = result.GroupType
	data.TeamId = result.TeamId
	data.InheritedFromId = result.InheritedFromId
	data.ParentId = result.ParentId
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *ProjectLabelsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ProjectLabelsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	labels, err := allProjectLabels(ctx, *d.client, data.IncludeArchived.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list project labels: %s", err))
		return
	}
	data.Id = types.StringValue("project_labels")
	data.Labels = []ProjectLabelLookupModel{}
	for _, label := range filterProjectLabels(labels, data.Name, data.ParentId, data.TeamId, data.IsGroup, data.WorkspaceOnly) {
		var model ProjectLabelLookupModel
		setProjectLabelLookupState(&model, label)
		data.Labels = append(data.Labels, model)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func allProjectLabels(ctx context.Context, client graphql.Client, includeArchived bool) ([]ProjectLabelLookup, error) {
	labels := []ProjectLabelLookup{}
	var after *string
	for {
		response, err := listProjectLabels(ctx, client, after, includeArchived)
		if err != nil {
			return nil, err
		}
		for _, node := range response.ProjectLabels.Nodes {
			labels = append(labels, node.ProjectLabelLookup)
		}
		page := response.ProjectLabels.PageInfo
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

func filterProjectLabels(labels []ProjectLabelLookup, name, parent, team types.String, isGroup, workspaceOnly types.Bool) []ProjectLabelLookup {
	result := []ProjectLabelLookup{}
	for _, label := range labels {
		if workspaceOnly.ValueBool() && label.Team != nil {
			continue
		}
		if !team.IsNull() && (label.Team == nil || label.Team.Id != team.ValueString()) {
			continue
		}
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

type ProjectLabelLookupModel struct {
	Id              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	Color           types.String `tfsdk:"color"`
	IsGroup         types.Bool   `tfsdk:"is_group"`
	GroupType       types.String `tfsdk:"group_type"`
	TeamId          types.String `tfsdk:"team_id"`
	InheritedFromId types.String `tfsdk:"inherited_from_id"`
	ParentId        types.String `tfsdk:"parent_id"`
}

func setProjectLabelLookupState(data *ProjectLabelLookupModel, projectLabel ProjectLabelLookup) {
	data.Id = types.StringValue(projectLabel.Id)
	data.Name = types.StringValue(projectLabel.Name)
	data.Description = types.StringPointerValue(projectLabel.Description)
	data.Color = types.StringValue(projectLabel.Color)
	data.IsGroup = types.BoolValue(projectLabel.IsGroup)
	data.GroupType = types.StringNull()
	if projectLabel.GroupType != nil {
		data.GroupType = types.StringValue(string(*projectLabel.GroupType))
	}
	data.TeamId = types.StringNull()
	if projectLabel.Team != nil {
		data.TeamId = types.StringValue(projectLabel.Team.Id)
	}
	data.InheritedFromId = types.StringNull()
	if projectLabel.InheritedFrom != nil {
		data.InheritedFromId = types.StringValue(projectLabel.InheritedFrom.Id)
	}
	if projectLabel.Parent != nil {
		data.ParentId = types.StringValue(projectLabel.Parent.Id)
	} else {
		data.ParentId = types.StringNull()
	}
}
