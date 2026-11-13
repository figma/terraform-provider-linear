package provider

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type webhookTestClient struct {
	call func(*graphql.Request) (string, error)
}

func (c webhookTestClient) MakeRequest(_ context.Context, req *graphql.Request, resp *graphql.Response) error {
	payload, err := c.call(req)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(payload), resp.Data)
}

const webhookTestConfig = `{"id":"hook-id","label":"Audit","url":"https://example.com/hook","enabled":false,"allPublicTeams":true,"team":null,"resourceTypes":["Comment","Issue"],"secret":"must-not-enter-state"}`

func webhookTestModel(secret string) WebhookResourceModel {
	return WebhookResourceModel{Id: types.StringValue("hook-id"), Label: types.StringValue("Audit"), Url: types.StringValue("https://example.com/hook"), Enabled: types.BoolValue(false), TeamId: types.StringNull(), AllPublicTeams: types.BoolValue(true), ResourceTypes: types.SetValueMust(types.StringType, []attr.Value{types.StringValue("Issue"), types.StringValue("Comment")}), SecretWO: types.StringValue(secret), SecretWOVersion: types.Int64Value(1)}
}

func webhookTestInputs(t *testing.T, schema rschema.Schema, model WebhookResourceModel) (tfsdk.Plan, tfsdk.Config) {
	t.Helper()
	rawConfig := tfsdk.State{Schema: schema}
	if d := rawConfig.Set(context.Background(), &model); d.HasError() {
		t.Fatal(d)
	}
	config := tfsdk.Config{Schema: schema, Raw: rawConfig.Raw}
	model.SecretWO = types.StringNull()
	plan := tfsdk.Plan{Schema: schema}
	if d := plan.Set(context.Background(), &model); d.HasError() {
		t.Fatal(d)
	}
	return plan, config
}

func TestWebhookSecretLifecycle(t *testing.T) {
	ctx := context.Background()
	var lastInput map[string]any
	var client graphql.Client = webhookTestClient{call: func(req *graphql.Request) (string, error) {
		if strings.Contains(req.Query, "secret") {
			t.Fatal("GraphQL selections must never retrieve secrets")
		}
		raw, _ := json.Marshal(req.Variables)
		var vars map[string]any
		_ = json.Unmarshal(raw, &vars)
		if vars["input"] != nil {
			lastInput = vars["input"].(map[string]any)
		}
		switch req.OpName {
		case "createWebhook":
			return `{"webhookCreate":{"success":true,"webhook":` + webhookTestConfig + `}}`, nil
		case "updateWebhook":
			return `{"webhookUpdate":{"success":true,"webhook":` + webhookTestConfig + `}}`, nil
		case "getWebhook":
			return `{"webhook":` + webhookTestConfig + `}`, nil
		}
		t.Fatalf("unexpected request %s", req.OpName)
		return "", nil
	}}
	r := &WebhookResource{client: &client}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	if !sr.Schema.Attributes["secret_wo"].IsSensitive() || !sr.Schema.Attributes["secret_wo"].IsWriteOnly() {
		t.Fatal("signing secret must be sensitive and write-only")
	}
	if _, ok := sr.Schema.Attributes["secret"]; ok {
		t.Fatal("stateful secret argument must not be exposed")
	}
	model := webhookTestModel("configured-secret")
	model.Id = types.StringUnknown()
	plan, config := webhookTestInputs(t, sr.Schema, model)
	created := resource.CreateResponse{State: tfsdk.State{Schema: sr.Schema}}
	r.Create(ctx, resource.CreateRequest{Plan: plan, Config: config}, &created)
	if created.Diagnostics.HasError() {
		t.Fatal(created.Diagnostics)
	}
	if lastInput["secret"] != "configured-secret" || lastInput["enabled"] != false {
		t.Fatal("create must read the secret from config and preserve the disabled state")
	}
	if _, ok := lastInput["teamId"]; ok {
		t.Fatal("all-public subscription must omit teamId")
	}
	var got WebhookResourceModel
	if d := created.State.Get(ctx, &got); d.HasError() {
		t.Fatal(d)
	}
	if !got.SecretWO.IsNull() || got.SecretWOVersion.ValueInt64() != 1 {
		t.Fatal("state must contain only the rotation version")
	}
	read := resource.ReadResponse{State: created.State}
	r.Read(ctx, resource.ReadRequest{State: created.State}, &read)
	if read.Diagnostics.HasError() {
		t.Fatal(read.Diagnostics)
	}
	if d := read.State.Get(ctx, &got); d.HasError() {
		t.Fatal(d)
	}
	if !got.SecretWO.IsNull() {
		t.Fatal("refresh must not retrieve signing secrets")
	}
	for _, tc := range []struct {
		name    string
		secret  types.String
		version types.Int64
		sent    bool
	}{
		{"unrelated update", types.StringValue("configured-secret"), types.Int64Value(1), false},
		{"secret changed without version", types.StringValue("rotated-secret"), types.Int64Value(1), false},
		{"explicit rotation", types.StringValue("rotated-secret"), types.Int64Value(2), true},
		{"stop managing secret", types.StringNull(), types.Int64Null(), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			model = got
			model.Label = types.StringValue("Updated")
			model.SecretWO, model.SecretWOVersion = tc.secret, tc.version
			plan, config = webhookTestInputs(t, sr.Schema, model)
			updated := resource.UpdateResponse{State: read.State}
			r.Update(ctx, resource.UpdateRequest{State: read.State, Plan: plan, Config: config}, &updated)
			if updated.Diagnostics.HasError() {
				t.Fatal(updated.Diagnostics)
			}
			secret, sent := lastInput["secret"]
			if sent != tc.sent || (sent && secret != tc.secret.ValueString()) {
				t.Fatal("update sent an unexpected secret")
			}
			if _, ok := lastInput["teamId"]; ok {
				t.Fatal("scope cannot be updated in place")
			}
			var saved WebhookResourceModel
			if d := updated.State.Get(ctx, &saved); d.HasError() {
				t.Fatal(d)
			}
			if !saved.SecretWO.IsNull() || !saved.SecretWOVersion.Equal(tc.version) {
				t.Fatal("update must persist only the rotation version")
			}
		})
	}
	imported := webhookTestModel("")
	imported.SecretWO, imported.SecretWOVersion = types.StringNull(), types.Int64Null()
	importedState := tfsdk.State{Schema: sr.Schema}
	if d := importedState.Set(ctx, &imported); d.HasError() {
		t.Fatal(d)
	}
	importedRead := resource.ReadResponse{State: importedState}
	r.Read(ctx, resource.ReadRequest{State: importedState}, &importedRead)
	if importedRead.Diagnostics.HasError() {
		t.Fatal(importedRead.Diagnostics)
	}
	if d := importedRead.State.Get(ctx, &got); d.HasError() {
		t.Fatal(d)
	}
	if !got.SecretWO.IsNull() || !got.SecretWOVersion.IsNull() {
		t.Fatal("import must not recover secret configuration")
	}
}

func TestWebhookErrorsDoNotDiscloseSecrets(t *testing.T) {
	ctx := context.Background()
	var client graphql.Client = webhookTestClient{call: func(*graphql.Request) (string, error) {
		return "", errors.New("request failed: secret=private-value https://example.com/?token=private-value")
	}}
	r := &WebhookResource{client: &client}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	model := webhookTestModel("private-value")
	plan, config := webhookTestInputs(t, sr.Schema, model)
	response := resource.CreateResponse{State: tfsdk.State{Schema: sr.Schema}}
	r.Create(ctx, resource.CreateRequest{Plan: plan, Config: config}, &response)
	if !response.Diagnostics.HasError() {
		t.Fatal("API failure must surface a diagnostic")
	}
	for _, diagnostic := range response.Diagnostics {
		if strings.Contains(diagnostic.Detail(), "private-value") {
			t.Fatal("API error disclosed credentials")
		}
	}
}

func TestWebhookPaginationAndSecretFreeLookups(t *testing.T) {
	ctx := context.Background()
	var one datasource.SchemaResponse
	(&WebhookDataSource{}).Schema(ctx, datasource.SchemaRequest{}, &one)
	if _, ok := one.Schema.Attributes["secret"]; ok {
		t.Fatal("lookup must not expose secrets")
	}
	var many datasource.SchemaResponse
	(&WebhooksDataSource{}).Schema(ctx, datasource.SchemaRequest{}, &many)
	nested := many.Schema.Attributes["webhooks"].(dschema.ListNestedAttribute)
	if _, ok := nested.NestedObject.Attributes["secret"]; ok {
		t.Fatal("collection must not expose secrets")
	}
	calls := 0
	var client graphql.Client = webhookTestClient{call: func(req *graphql.Request) (string, error) {
		calls++
		if strings.Contains(req.Query, "secret") {
			t.Fatal("lookup requested signing secrets")
		}
		raw, _ := json.Marshal(req.Variables)
		var vars map[string]any
		_ = json.Unmarshal(raw, &vars)
		if calls == 1 {
			return `{"webhooks":{"nodes":[` + webhookTestConfig + `],"pageInfo":{"hasNextPage":true,"endCursor":"next"}}}`, nil
		}
		if vars["after"] != "next" {
			t.Fatal("missing cursor")
		}
		return `{"webhooks":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":null}}}`, nil
	}}
	hooks, err := allWebhooks(ctx, client)
	if err != nil || calls != 2 || len(hooks) != 1 {
		t.Fatalf("pagination failed: %d %d %v", calls, len(hooks), err)
	}
	got := webhookLookupValues(hooks[0])
	if got.ResourceTypes.IsNull() || got.Enabled.ValueBool() {
		t.Fatal("lookup lost configuration")
	}
}

func TestWebhookScopeAndURLValidation(t *testing.T) {
	ctx := context.Background()
	r := &WebhookResource{}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	for _, tc := range []struct {
		team    types.String
		all     bool
		invalid bool
	}{
		{types.StringNull(), true, false},
		{types.StringValue("team"), false, false},
		{types.StringNull(), false, true},
		{types.StringValue("team"), true, true},
	} {
		model := webhookTestModel("test-secret")
		model.TeamId, model.AllPublicTeams = tc.team, types.BoolValue(tc.all)
		state := tfsdk.State{Schema: sr.Schema}
		if d := state.Set(ctx, &model); d.HasError() {
			t.Fatal(d)
		}
		var resp resource.ValidateConfigResponse
		r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: sr.Schema, Raw: state.Raw}}, &resp)
		if resp.Diagnostics.HasError() != tc.invalid {
			t.Fatalf("unexpected scope validation: %v", resp.Diagnostics)
		}
	}
	for _, value := range []string{"http://example.com", "https://user:password@example.com", "https://example.com/#fragment"} {
		var resp validator.StringResponse
		webhookURLValidator{}.ValidateString(ctx, validator.StringRequest{Path: path.Root("url"), ConfigValue: types.StringValue(value)}, &resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("invalid URL accepted")
		}
	}
}

func TestWebhookSecretPairValidation(t *testing.T) {
	ctx := context.Background()
	r := &WebhookResource{}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	for _, tc := range []struct {
		name    string
		secret  types.String
		version types.Int64
		invalid bool
	}{
		{"missing version", types.StringValue("secret"), types.Int64Null(), true},
		{"missing secret", types.StringNull(), types.Int64Value(1), true},
		{"unmanaged", types.StringNull(), types.Int64Null(), false},
		{"unknown secret", types.StringUnknown(), types.Int64Value(1), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			model := webhookTestModel("")
			model.SecretWO, model.SecretWOVersion = tc.secret, tc.version
			_, config := webhookTestInputs(t, sr.Schema, model)
			var resp resource.ValidateConfigResponse
			r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: config}, &resp)
			if resp.Diagnostics.HasError() != tc.invalid {
				t.Fatalf("unexpected secret validation: %v", resp.Diagnostics)
			}
		})
	}
}
