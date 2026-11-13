package provider

import (
	"context"
	"fmt"
	"sort"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewWebhookDataSource() datasource.DataSource  { return &WebhookDataSource{} }
func NewWebhooksDataSource() datasource.DataSource { return &WebhooksDataSource{} }

type WebhookDataSource struct{ client *graphql.Client }
type WebhooksDataSource struct{ WebhookDataSource }
type WebhookLookupModel struct {
	Id             types.String `tfsdk:"id"`
	Label          types.String `tfsdk:"label"`
	Url            types.String `tfsdk:"url"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	TeamId         types.String `tfsdk:"team_id"`
	AllPublicTeams types.Bool   `tfsdk:"all_public_teams"`
	ResourceTypes  types.Set    `tfsdk:"resource_types"`
}
type WebhooksDataSourceModel struct {
	Id             types.String         `tfsdk:"id"`
	Label          types.String         `tfsdk:"label"`
	TeamId         types.String         `tfsdk:"team_id"`
	Enabled        types.Bool           `tfsdk:"enabled"`
	AllPublicTeams types.Bool           `tfsdk:"all_public_teams"`
	Webhooks       []WebhookLookupModel `tfsdk:"webhooks"`
}

func webhookLookupAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id":               schema.StringAttribute{Computed: true, MarkdownDescription: "Webhook UUID."},
		"label":            schema.StringAttribute{Computed: true, MarkdownDescription: "Subscription label."},
		"url":              schema.StringAttribute{Computed: true, MarkdownDescription: "Receiver URL. Null for OAuth application webhooks."},
		"enabled":          schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether event delivery is enabled."},
		"team_id":          schema.StringAttribute{Computed: true, MarkdownDescription: "Subscribed team UUID, when team-scoped."},
		"all_public_teams": schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether all public teams are subscribed."},
		"resource_types":   schema.SetAttribute{Computed: true, ElementType: types.StringType, MarkdownDescription: "Subscribed event resource names."},
	}
}
func (d *WebhookDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}
func (d *WebhooksDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhooks"
}
func (d *WebhookDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	a := webhookLookupAttributes()
	a["id"] = schema.StringAttribute{Required: true, MarkdownDescription: "Exact webhook UUID.", Validators: []validator.String{stringvalidator.RegexMatches(uuidRegex(), "must be a UUID")}}
	resp.Schema = schema.Schema{MarkdownDescription: "Read webhook configuration by UUID. Requires API-management permission. Signing secrets are never queried or returned.", Attributes: a}
}
func (d *WebhooksDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "List webhook subscriptions with cursor pagination. Filters match exactly and combine with AND. Signing secrets are never queried or returned.", Attributes: map[string]schema.Attribute{
		"id":               schema.StringAttribute{Computed: true, MarkdownDescription: "Collection identifier."},
		"label":            schema.StringAttribute{Optional: true, MarkdownDescription: "Exact label filter."},
		"team_id":          schema.StringAttribute{Optional: true, MarkdownDescription: "Subscribed team UUID filter.", Validators: []validator.String{stringvalidator.RegexMatches(uuidRegex(), "must be a UUID")}},
		"enabled":          schema.BoolAttribute{Optional: true, MarkdownDescription: "Filter by delivery status."},
		"all_public_teams": schema.BoolAttribute{Optional: true, MarkdownDescription: "Filter by all-public-team subscription status."},
		"webhooks":         schema.ListNestedAttribute{Computed: true, MarkdownDescription: "Matching subscriptions, sorted by UUID.", NestedObject: schema.NestedAttributeObject{Attributes: webhookLookupAttributes()}},
	}}
}
func (d *WebhookDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *WebhookDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data WebhookLookupModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := getWebhook(ctx, *d.client, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Webhook lookup failed", "Unable to read this webhook. Check the UUID and API-management permission. API error details are omitted to protect webhook credentials.")
		return
	}
	data = webhookLookupValues(result.Webhook.WebhookConfig)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
func (d *WebhooksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data WebhooksDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	webhooks, err := allWebhooks(ctx, *d.client)
	if err != nil {
		resp.Diagnostics.AddError("Webhook lookup failed", "Unable to list webhooks. Check API-management permission. API error details are omitted to protect webhook credentials.")
		return
	}
	data.Id = types.StringValue("webhooks")
	data.Webhooks = []WebhookLookupModel{}
	for _, config := range webhooks {
		if !data.Label.IsNull() && (config.Label == nil || *config.Label != data.Label.ValueString()) {
			continue
		}
		if !data.TeamId.IsNull() && (config.Team == nil || config.Team.Id != data.TeamId.ValueString()) {
			continue
		}
		if !data.Enabled.IsNull() && config.Enabled != data.Enabled.ValueBool() {
			continue
		}
		if !data.AllPublicTeams.IsNull() && config.AllPublicTeams != data.AllPublicTeams.ValueBool() {
			continue
		}
		data.Webhooks = append(data.Webhooks, webhookLookupValues(config))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
func webhookLookupValues(config WebhookConfig) WebhookLookupModel {
	var data WebhookResourceModel
	applyWebhookConfig(&data, config)
	return WebhookLookupModel{Id: data.Id, Label: data.Label, Url: data.Url, Enabled: data.Enabled, TeamId: data.TeamId, AllPublicTeams: data.AllPublicTeams, ResourceTypes: data.ResourceTypes}
}
func allWebhooks(ctx context.Context, client graphql.Client) ([]WebhookConfig, error) {
	result := []WebhookConfig{}
	var after *string
	for {
		response, err := listWebhooks(ctx, client, after)
		if err != nil {
			return nil, err
		}
		for _, node := range response.Webhooks.Nodes {
			result = append(result, node.WebhookConfig)
		}
		page := response.Webhooks.PageInfo
		if !page.HasNextPage {
			break
		}
		if page.EndCursor == nil || *page.EndCursor == "" || (after != nil && *after == *page.EndCursor) {
			return nil, fmt.Errorf("Linear returned an invalid pagination cursor")
		}
		after = page.EndCursor
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Id < result[j].Id })
	return result, nil
}
