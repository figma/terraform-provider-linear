package provider

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewWebhookResource() resource.Resource { return &WebhookResource{} }

type WebhookResource struct{ client *graphql.Client }
type WebhookResourceModel struct {
	Id              types.String `tfsdk:"id"`
	Label           types.String `tfsdk:"label"`
	Url             types.String `tfsdk:"url"`
	Enabled         types.Bool   `tfsdk:"enabled"`
	TeamId          types.String `tfsdk:"team_id"`
	AllPublicTeams  types.Bool   `tfsdk:"all_public_teams"`
	ResourceTypes   types.Set    `tfsdk:"resource_types"`
	SecretWO        types.String `tfsdk:"secret_wo"`
	SecretWOVersion types.Int64  `tfsdk:"secret_wo_version"`
}

func (r *WebhookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}
func (r *WebhookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Linear webhook subscription for one team or all public teams. Requires workspace API-management permission and relevant team permissions. Import by UUID. See WEBHOOKS.md in the provider repository for secret handling and lifecycle details.", Attributes: map[string]schema.Attribute{
		"id":                schema.StringAttribute{Computed: true, MarkdownDescription: "Webhook UUID.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"label":             schema.StringAttribute{Optional: true, MarkdownDescription: "Human-readable description of the subscription."},
		"url":               schema.StringAttribute{Required: true, MarkdownDescription: "Public HTTPS receiver URL. Do not embed credentials in this URL.", Validators: []validator.String{webhookURLValidator{}}},
		"enabled":           schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true), MarkdownDescription: "Whether to deliver events. Defaults to true."},
		"team_id":           schema.StringAttribute{Optional: true, MarkdownDescription: "Team UUID. Choose this or all_public_teams = true. Scope changes replace the subscription.", Validators: []validator.String{stringvalidator.RegexMatches(uuidRegex(), "must be a UUID")}, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
		"all_public_teams":  schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false), MarkdownDescription: "Subscribe to all public teams, including future public teams. Scope changes replace the subscription.", PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}},
		"resource_types":    schema.SetAttribute{Required: true, ElementType: types.StringType, MarkdownDescription: "Event resource names, such as Issue and Comment. Treated as an unordered set. Available event types and permissions are validated by Linear.", Validators: []validator.Set{setvalidator.SizeAtLeast(1), setvalidator.ValueStringsAre(stringvalidator.UTF8LengthAtLeast(1))}},
		"secret_wo":         schema.StringAttribute{Optional: true, Sensitive: true, WriteOnly: true, MarkdownDescription: "HMAC-SHA256 signing secret. Requires OpenTofu 1.11+ or Terraform 1.11.1+ for sensitive ephemeral variables. Read only from configuration and never stored in resource state. Supply through an ephemeral, sensitive variable to also exclude its value from saved plans. Configure secret_wo_version alongside it and change the version to rotate. Omit both to leave the secret unmanaged; Linear generates one on creation. Refresh/import never retrieve secrets, so external rotations are not detected.", Validators: []validator.String{stringvalidator.UTF8LengthAtLeast(1)}},
		"secret_wo_version": schema.Int64Attribute{Optional: true, MarkdownDescription: "Positive rotation version, required with secret_wo. Change this value whenever changing the signing secret; changing only secret_wo does not trigger an update. Removing both arguments leaves the remote secret untouched.", Validators: []validator.Int64{int64validator.AtLeast(1)}},
	}}
}

func (r *WebhookResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*graphql.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *graphql.Client, got %T.", req.ProviderData))
		return
	}
	r.client = client
}

func (r *WebhookResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data WebhookResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !data.TeamId.IsUnknown() && !data.AllPublicTeams.IsUnknown() && (!data.TeamId.IsNull()) == data.AllPublicTeams.ValueBool() {
		resp.Diagnostics.AddAttributeError(path.Root("team_id"), "Invalid webhook scope", "Specify exactly one of team_id or all_public_teams = true.")
	}
	if !data.SecretWO.IsUnknown() && !data.SecretWOVersion.IsUnknown() && data.SecretWO.IsNull() != data.SecretWOVersion.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("secret_wo_version"), "Incomplete webhook secret configuration", "Configure secret_wo and secret_wo_version together, or omit both to leave the signing secret unmanaged.")
	}
}

func (r *WebhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WebhookResourceModel
	var secret types.String
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("secret_wo"), &secret)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if secret.IsNull() != data.SecretWOVersion.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("secret_wo_version"), "Incomplete webhook secret configuration", "Configure secret_wo and secret_wo_version together, or omit both.")
		return
	}
	var resourceTypes []string
	resp.Diagnostics.Append(data.ResourceTypes.ElementsAs(ctx, &resourceTypes, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := createWebhook(ctx, *r.client, WebhookCreateInput{Label: data.Label.ValueStringPointer(), Url: data.Url.ValueString(), Enabled: data.Enabled.ValueBool(), TeamId: data.TeamId.ValueStringPointer(), AllPublicTeams: data.AllPublicTeams.ValueBool(), ResourceTypes: resourceTypes, Secret: secret.ValueStringPointer()})
	if err != nil {
		resp.Diagnostics.AddError("Webhook create failed", "Linear could not create the subscription. Check API-management permissions, team scope, destination URL and event types. API error details are omitted to protect webhook credentials.")
		return
	}
	if !result.WebhookCreate.Success {
		resp.Diagnostics.AddError("Webhook create failed", "Linear reported an unsuccessful creation.")
		return
	}
	applyWebhookConfig(&data, result.WebhookCreate.Webhook.WebhookConfig)
	data.SecretWO = types.StringNull()
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WebhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WebhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := getWebhook(ctx, *r.client, data.Id.ValueString())
	if err != nil {
		if isWebhookNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Webhook read failed", "Linear could not read the subscription. Check that it still exists and that you have API-management permission. API error details are omitted to protect webhook credentials.")
		return
	}
	if result.Webhook.Url == nil {
		resp.Diagnostics.AddError("Unsupported webhook", "This subscription has no destination URL. Manage OAuth application webhooks through the application configuration.")
		return
	}
	applyWebhookConfig(&data, result.Webhook.WebhookConfig)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WebhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, prior WebhookResourceModel
	var secret types.String
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("secret_wo"), &secret)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if secret.IsNull() != data.SecretWOVersion.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("secret_wo_version"), "Incomplete webhook secret configuration", "Configure secret_wo and secret_wo_version together, or omit both.")
		return
	}
	var resourceTypes []string
	resp.Diagnostics.Append(data.ResourceTypes.ElementsAs(ctx, &resourceTypes, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	input := WebhookUpdateInput{Label: data.Label.ValueStringPointer(), Url: data.Url.ValueString(), Enabled: data.Enabled.ValueBool(), ResourceTypes: resourceTypes}
	if !data.SecretWOVersion.IsNull() && !data.SecretWOVersion.Equal(prior.SecretWOVersion) {
		input.Secret = secret.ValueStringPointer()
	}
	result, err := updateWebhook(ctx, *r.client, data.Id.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Webhook update failed", "Linear could not update the subscription. Check API-management permissions, destination URL and event types. API error details are omitted to protect webhook credentials.")
		return
	}
	if !result.WebhookUpdate.Success {
		resp.Diagnostics.AddError("Webhook update failed", "Linear reported an unsuccessful update.")
		return
	}
	applyWebhookConfig(&data, result.WebhookUpdate.Webhook.WebhookConfig)
	data.SecretWO = types.StringNull()
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WebhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WebhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := deleteWebhook(ctx, *r.client, data.Id.ValueString())
	if err != nil {
		if isWebhookNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Webhook delete failed", "Linear could not delete the subscription. Check API-management permission. API error details are omitted to protect webhook credentials.")
		return
	}
	if !result.WebhookDelete.Success {
		resp.Diagnostics.AddError("Webhook delete failed", "Linear reported an unsuccessful deletion.")
	}
}

func (r *WebhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyWebhookConfig(data *WebhookResourceModel, config WebhookConfig) {
	data.Id = types.StringValue(config.Id)
	data.Label = types.StringPointerValue(config.Label)
	data.Url = types.StringPointerValue(config.Url)
	data.Enabled = types.BoolValue(config.Enabled)
	data.AllPublicTeams = types.BoolValue(config.AllPublicTeams)
	data.TeamId = types.StringNull()
	if config.Team != nil {
		data.TeamId = types.StringValue(config.Team.Id)
	}
	values := []attr.Value{}
	for _, value := range config.ResourceTypes {
		values = append(values, types.StringValue(value))
	}
	data.ResourceTypes = types.SetValueMust(types.StringType, values)
}

func isWebhookNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "Entity not found: Webhook")
}

type webhookURLValidator struct{}

func (webhookURLValidator) Description(context.Context) string {
	return "A public HTTPS URL without embedded credentials."
}
func (v webhookURLValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (webhookURLValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	parsed, err := url.Parse(req.ConfigValue.ValueString())
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil || parsed.Fragment != "" {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid webhook URL", "Use an absolute HTTPS URL without embedded credentials or a fragment.")
	}
}
