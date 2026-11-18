package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func teamSettingsTestState(t *testing.T) (*TeamResource, *teamRecordingClient, tfsdk.State) {
	t.Helper()
	workflowStates := make([]map[string]interface{}, 0, 5)
	for _, kind := range []string{"backlog", "unstarted", "started", "completed", "canceled"} {
		workflowStates = append(workflowStates, map[string]interface{}{
			"id": kind + "-id", "name": kind, "type": kind, "position": 0,
			"color": "#ffffff", "team": map[string]string{"id": testTeamUUID},
		})
	}
	workflowPayload, err := json.Marshal(map[string]interface{}{"workflowStates": map[string]interface{}{"nodes": workflowStates}})
	if err != nil {
		t.Fatal(err)
	}
	team := fmt.Sprintf(`{"id":%q,"key":"TEST","name":"Test team","private":true,"timezone":"Etc/GMT","setIssueSortOrderOnStateChange":"first","cyclesEnabled":true,"cycleDuration":2,"cycleIssueAutoAssignStarted":true,"issueEstimationType":"linear","issueEstimationAllowZero":true,"defaultIssueEstimate":1}`, testTeamUUID)
	r, client, state := teamResourceForUnitTest(t, map[string]string{
		"getTeamByID":           fmt.Sprintf(`{"administrableTeams":{"nodes":[%s]}}`, team),
		"getTeamWorkflowStates": string(workflowPayload),
		"updateTeam":            fmt.Sprintf(`{"teamUpdate":{"team":%s}}`, team),
		"updateWorkflowState":   `{"workflowStateUpdate":{"workflowState":{"id":"started-id","name":"Working","position":0,"color":"#ffffff"}}}`,
	})
	if diags := state.SetAttribute(context.Background(), path.Root("id"), testTeamUUID); diags.HasError() {
		t.Fatal(diags)
	}
	response := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	return r, client, response.State
}

func TestTeamIgnoredSettingsSurviveInheritedImportPlan(t *testing.T) {
	r, _, state := teamSettingsTestState(t)
	ctx := context.Background()
	for name, value := range map[string]interface{}{
		"parent_id":                 "ab69b823-f187-444d-8848-782045b86656",
		"inherit_issue_estimation":  true,
		"inherit_workflow_statuses": true,
	} {
		if diags := state.SetAttribute(ctx, path.Root(name), value); diags.HasError() {
			t.Fatal(diags)
		}
	}
	request := resource.ModifyPlanRequest{
		Config: tfsdk.Config{Schema: state.Schema, Raw: state.Raw},
		Plan:   tfsdk.Plan{Schema: state.Schema, Raw: state.Raw},
		State:  state,
	}
	response := resource.ModifyPlanResponse{Plan: request.Plan}
	r.ModifyPlan(ctx, request, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("unchanged imported settings should plan: %v", response.Diagnostics)
	}
}

func TestTeamChangedWorkflowPlansNewPosition(t *testing.T) {
	r, _, state := teamSettingsTestState(t)
	ctx := context.Background()
	var before TeamResourceModel
	if diags := state.Get(ctx, &before); diags.HasError() {
		t.Fatal(diags)
	}
	started := before.StartedWorkflowState.Attributes()
	started["name"] = types.StringValue("Working")
	plan := tfsdk.Plan{Schema: state.Schema, Raw: state.Raw}
	if diags := plan.SetAttribute(ctx, path.Root("started_workflow_state"), types.ObjectValueMust(workflowStateAttrTypes, started)); diags.HasError() {
		t.Fatal(diags)
	}
	request := resource.ModifyPlanRequest{
		Config: tfsdk.Config{Schema: state.Schema, Raw: plan.Raw},
		Plan:   plan,
		State:  state,
	}
	response := resource.ModifyPlanResponse{Plan: plan}
	r.ModifyPlan(ctx, request, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	var after TeamResourceModel
	if diags := response.Plan.Get(ctx, &after); diags.HasError() {
		t.Fatal(diags)
	}
	if !after.StartedWorkflowState.Attributes()["position"].IsUnknown() {
		t.Fatal("edited workflow position must be recomputed by Linear")
	}
	if !after.UnstartedWorkflowState.Equal(before.UnstartedWorkflowState) {
		t.Fatal("unchanged workflow state should retain its planned value")
	}
}

func TestTeamDisablingWorkflowInheritancePlansLocalStates(t *testing.T) {
	for _, test := range []struct {
		name           string
		copyState      bool
		configureState bool
	}{
		{name: "omitted"},
		{name: "ignored state values", copyState: true},
		{name: "explicit local state", configureState: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			r, _, state := teamSettingsTestState(t)
			ctx := context.Background()
			if diags := state.SetAttribute(ctx, path.Root("parent_id"), "ab69b823-f187-444d-8848-782045b86656"); diags.HasError() {
				t.Fatal(diags)
			}
			if diags := state.SetAttribute(ctx, path.Root("inherit_workflow_statuses"), true); diags.HasError() {
				t.Fatal(diags)
			}
			plan := tfsdk.Plan{Schema: state.Schema, Raw: state.Raw}
			configValues := tfsdk.Plan{Schema: state.Schema, Raw: state.Raw}
			for _, values := range []*tfsdk.Plan{&plan, &configValues} {
				if diags := values.SetAttribute(ctx, path.Root("inherit_workflow_statuses"), false); diags.HasError() {
					t.Fatal(diags)
				}
			}
			workflowNames := []string{"backlog_workflow_state", "unstarted_workflow_state", "started_workflow_state", "completed_workflow_state", "canceled_workflow_state"}
			if !test.copyState {
				for _, name := range workflowNames {
					if diags := configValues.SetAttribute(ctx, path.Root(name), types.ObjectNull(workflowStateAttrTypes)); diags.HasError() {
						t.Fatal(diags)
					}
				}
			}
			if test.configureState {
				var before TeamResourceModel
				if diags := state.Get(ctx, &before); diags.HasError() {
					t.Fatal(diags)
				}
				started := before.StartedWorkflowState.Attributes()
				started["name"] = types.StringValue("Local working")
				configured := types.ObjectValueMust(workflowStateAttrTypes, started)
				if diags := configValues.SetAttribute(ctx, path.Root("started_workflow_state"), configured); diags.HasError() {
					t.Fatal(diags)
				}
				if diags := plan.SetAttribute(ctx, path.Root("started_workflow_state"), configured); diags.HasError() {
					t.Fatal(diags)
				}
			}
			request := resource.ModifyPlanRequest{
				Config: tfsdk.Config{Schema: state.Schema, Raw: configValues.Raw},
				Plan:   plan,
				State:  state,
			}
			response := resource.ModifyPlanResponse{Plan: plan}
			r.ModifyPlan(ctx, request, &response)
			if response.Diagnostics.HasError() {
				t.Fatal(response.Diagnostics)
			}
			var after TeamResourceModel
			if diags := response.Plan.Get(ctx, &after); diags.HasError() {
				t.Fatal(diags)
			}
			for name, workflow := range map[string]types.Object{
				"backlog_workflow_state":   after.BacklogWorkflowState,
				"unstarted_workflow_state": after.UnstartedWorkflowState,
				"started_workflow_state":   after.StartedWorkflowState,
				"completed_workflow_state": after.CompletedWorkflowState,
				"canceled_workflow_state":  after.CanceledWorkflowState,
			} {
				if test.configureState && name == "started_workflow_state" {
					if workflow.IsUnknown() || !workflow.Attributes()["id"].IsUnknown() || !workflow.Attributes()["position"].IsUnknown() {
						t.Errorf("%s should retain configured fields with unknown local ID and position", name)
					}
				} else if !workflow.IsUnknown() {
					t.Errorf("%s should be unknown until local state is read", name)
				}
			}
		})
	}
}

func TestTeamDisablingEstimationInheritancePlansLocalSettings(t *testing.T) {
	r, _, state := teamSettingsTestState(t)
	ctx := context.Background()
	if diags := state.SetAttribute(ctx, path.Root("parent_id"), "ab69b823-f187-444d-8848-782045b86656"); diags.HasError() {
		t.Fatal(diags)
	}
	if diags := state.SetAttribute(ctx, path.Root("inherit_issue_estimation"), true); diags.HasError() {
		t.Fatal(diags)
	}
	plan := tfsdk.Plan{Schema: state.Schema, Raw: state.Raw}
	config := tfsdk.Plan{Schema: state.Schema, Raw: state.Raw}
	for _, values := range []*tfsdk.Plan{&plan, &config} {
		if diags := values.SetAttribute(ctx, path.Root("inherit_issue_estimation"), false); diags.HasError() {
			t.Fatal(diags)
		}
	}
	if diags := config.SetAttribute(ctx, path.Root("estimation"), types.ObjectNull(estimationAttrTypes)); diags.HasError() {
		t.Fatal(diags)
	}
	request := resource.ModifyPlanRequest{
		Config: tfsdk.Config{Schema: state.Schema, Raw: config.Raw},
		Plan:   plan,
		State:  state,
	}
	response := resource.ModifyPlanResponse{Plan: plan}
	r.ModifyPlan(ctx, request, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	var after TeamResourceModel
	if diags := response.Plan.Get(ctx, &after); diags.HasError() {
		t.Fatal(diags)
	}
	if !after.Estimation.IsUnknown() {
		t.Fatal("omitted estimation must be unknown until local settings are read")
	}
}

func TestTeamIdentityUpdateLeavesSettingsUntouched(t *testing.T) {
	r, client, state := teamSettingsTestState(t)
	plan := tfsdk.Plan{Schema: state.Schema, Raw: state.Raw}
	if diags := plan.SetAttribute(context.Background(), path.Root("name"), "Renamed team"); diags.HasError() {
		t.Fatal(diags)
	}
	response := resource.UpdateResponse{State: state}
	r.Update(context.Background(), resource.UpdateRequest{State: state, Plan: plan}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	if got := len(client.requests); got != 3 {
		t.Fatalf("expected read, workflow read and team update only; got %d requests", got)
	}
	if client.requests[2].OpName != "updateTeam" {
		t.Fatalf("unexpected operation %s", client.requests[2].OpName)
	}
	encoded, err := json.Marshal(client.requests[2].Variables)
	if err != nil {
		t.Fatal(err)
	}
	var variables struct {
		Input map[string]interface{} `json:"input"`
	}
	if err := json.Unmarshal(encoded, &variables); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"cyclesEnabled", "cycleDuration", "cycleIssueAutoAssignStarted", "issueEstimationType", "issueEstimationAllowZero", "defaultIssueEstimate", "triageEnabled", "inheritIssueEstimation", "inheritWorkflowStatuses"} {
		if _, exists := variables.Input[name]; exists {
			t.Errorf("identity update sent %s: %s", name, encoded)
		}
	}
}

func TestTeamSettingUpdateCanSendFalseAndZero(t *testing.T) {
	r, client, state := teamSettingsTestState(t)
	ctx := context.Background()
	var before TeamResourceModel
	if diags := state.Get(ctx, &before); diags.HasError() {
		t.Fatal(diags)
	}
	after := before
	cycleValues := before.Cycles.Attributes()
	cycleValues["enabled"] = types.BoolValue(false)
	after.Cycles = types.ObjectValueMust(cyclesAttrTypes, cycleValues)
	estimationValues := before.Estimation.Attributes()
	estimationValues["allow_zero"] = types.BoolValue(false)
	estimationValues["default"] = types.Float64Value(0)
	after.Estimation = types.ObjectValueMust(estimationAttrTypes, estimationValues)
	if diags := update(ctx, r.client, before, &after); diags.HasError() {
		t.Fatal(diags)
	}
	encoded, err := json.Marshal(client.requests[2].Variables)
	if err != nil {
		t.Fatal(err)
	}
	var variables struct {
		Input map[string]interface{} `json:"input"`
	}
	if err := json.Unmarshal(encoded, &variables); err != nil {
		t.Fatal(err)
	}
	for name, expected := range map[string]interface{}{"cyclesEnabled": false, "issueEstimationAllowZero": false, "defaultIssueEstimate": float64(0)} {
		if variables.Input[name] != expected {
			t.Errorf("%s = %v, want %v in %s", name, variables.Input[name], expected, encoded)
		}
	}
}

func TestTeamParentUpdateSendsInheritanceSettings(t *testing.T) {
	r, client, state := teamSettingsTestState(t)
	ctx := context.Background()
	var before TeamResourceModel
	if diags := state.Get(ctx, &before); diags.HasError() {
		t.Fatal(diags)
	}
	after := before
	after.ParentId = types.StringValue("ab69b823-f187-444d-8848-782045b86656")
	if diags := update(ctx, r.client, before, &after); diags.HasError() {
		t.Fatal(diags)
	}
	encoded, err := json.Marshal(client.requests[2].Variables)
	if err != nil {
		t.Fatal(err)
	}
	var variables struct {
		Input map[string]interface{} `json:"input"`
	}
	if err := json.Unmarshal(encoded, &variables); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"inheritIssueEstimation", "inheritWorkflowStatuses"} {
		if value, exists := variables.Input[name]; !exists || value != false {
			t.Errorf("parent change must send %s=false: %s", name, encoded)
		}
	}
}

func TestTeamInheritedWorkflowUpdateMatchesRead(t *testing.T) {
	r, client, state := teamSettingsTestState(t)
	const parentID = "ab69b823-f187-444d-8848-782045b86656"
	ctx := context.Background()
	if diags := state.SetAttribute(ctx, path.Root("parent_id"), parentID); diags.HasError() {
		t.Fatal(diags)
	}
	if diags := state.SetAttribute(ctx, path.Root("inherit_workflow_statuses"), true); diags.HasError() {
		t.Fatal(diags)
	}
	client.responses["updateTeam"] = fmt.Sprintf(`{"teamUpdate":{"team":{"id":%q,"key":"TEST","name":"Renamed team","parent":{"id":%q},"inheritWorkflowStatuses":true,"timezone":"Etc/GMT","setIssueSortOrderOnStateChange":"first"}}}`, testTeamUUID, parentID)
	plan := tfsdk.Plan{Schema: state.Schema, Raw: state.Raw}
	if diags := plan.SetAttribute(ctx, path.Root("name"), "Renamed team"); diags.HasError() {
		t.Fatal(diags)
	}
	response := resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{State: state, Plan: plan}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	if len(client.requests) != 4 || client.requests[3].OpName != "getTeamWorkflowStates" || requestTeamID(t, client.requests[3]) != testTeamUUID {
		t.Fatalf("inherited workflows must use the child team's read view %s: %#v", testTeamUUID, client.requests)
	}
	var before, after TeamResourceModel
	if diags := state.Get(ctx, &before); diags.HasError() {
		t.Fatal(diags)
	}
	if diags := response.State.Get(ctx, &after); diags.HasError() {
		t.Fatal(diags)
	}
	if after.Name.ValueString() != "Renamed team" {
		t.Fatalf("updated name = %q, want Renamed team", after.Name.ValueString())
	}
	for name, pair := range map[string][2]types.Object{
		"backlog":   {before.BacklogWorkflowState, after.BacklogWorkflowState},
		"unstarted": {before.UnstartedWorkflowState, after.UnstartedWorkflowState},
		"started":   {before.StartedWorkflowState, after.StartedWorkflowState},
		"completed": {before.CompletedWorkflowState, after.CompletedWorkflowState},
		"canceled":  {before.CanceledWorkflowState, after.CanceledWorkflowState},
	} {
		if !pair[0].Equal(pair[1]) {
			t.Errorf("name-only update changed inherited %s workflow state: before=%v after=%v", name, pair[0], pair[1])
		}
	}
}

func TestTeamWorkflowUpdateOnlyWritesChangedState(t *testing.T) {
	r, client, state := teamSettingsTestState(t)
	ctx := context.Background()
	var before TeamResourceModel
	if diags := state.Get(ctx, &before); diags.HasError() {
		t.Fatal(diags)
	}
	started := before.StartedWorkflowState.Attributes()
	started["name"] = types.StringValue("Working")
	plan := tfsdk.Plan{Schema: state.Schema, Raw: state.Raw}
	if diags := plan.SetAttribute(ctx, path.Root("started_workflow_state"), types.ObjectValueMust(workflowStateAttrTypes, started)); diags.HasError() {
		t.Fatal(diags)
	}
	response := resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{State: state, Plan: plan}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	if len(client.requests) != 4 || client.requests[2].OpName != "updateTeam" || client.requests[3].OpName != "updateWorkflowState" {
		t.Fatalf("expected one team update and one workflow state update, got %#v", client.requests)
	}
	if got := requestTeamID(t, client.requests[3]); got != "started-id" {
		t.Fatalf("updated workflow state %q, want started-id", got)
	}
}

func TestTeamCreateLeavesOmittedSettingsToLinear(t *testing.T) {
	ctx := context.Background()
	workflowStates := make([]map[string]interface{}, 0, 5)
	for _, kind := range []string{"backlog", "unstarted", "started", "completed", "canceled"} {
		workflowStates = append(workflowStates, map[string]interface{}{
			"id": kind + "-id", "name": kind, "type": kind, "position": 0,
			"color": "#ffffff", "team": map[string]string{"id": testTeamUUID},
		})
	}
	workflowPayload, err := json.Marshal(map[string]interface{}{"workflowStates": map[string]interface{}{"nodes": workflowStates}})
	if err != nil {
		t.Fatal(err)
	}
	team := fmt.Sprintf(`{"id":%q,"key":"TEST","name":"Test team","timezone":"Etc/GMT","setIssueSortOrderOnStateChange":"first","cycleDuration":2,"issueEstimationType":"linear","defaultIssueEstimate":1}`, testTeamUUID)
	r, client, state := teamResourceForUnitTest(t, map[string]string{
		"createTeam":            fmt.Sprintf(`{"teamCreate":{"team":%s}}`, team),
		"updateTeam":            fmt.Sprintf(`{"teamUpdate":{"team":%s}}`, team),
		"getTeamWorkflowStates": string(workflowPayload),
	})
	plan := tfsdk.Plan{Schema: state.Schema, Raw: tftypes.NewValue(state.Schema.Type().TerraformType(ctx), nil)}
	data := TeamResourceModel{
		Key: types.StringValue("TEST"), Name: types.StringValue("Test team"),
		Private: types.BoolValue(false), Timezone: types.StringValue("Etc/GMT"),
		Description: types.StringNull(), ParentId: types.StringNull(),
		Icon: types.StringUnknown(), Color: types.StringUnknown(),
		EnableIssueHistoryGrouping: types.BoolValue(false),
		EnableIssueDefaultToBottom: types.BoolValue(false),
		AutoArchivePeriod:          types.Float64Value(0), AutoClosePeriod: types.Float64Value(0),
		Triage: types.ObjectValueMust(triageAttrTypes, map[string]attr.Value{
			"enabled": types.BoolValue(false), "require_priority": types.BoolValue(false),
		}),
		Cycles:                 types.ObjectUnknown(cyclesAttrTypes),
		Estimation:             types.ObjectUnknown(estimationAttrTypes),
		BacklogWorkflowState:   types.ObjectUnknown(workflowStateAttrTypes),
		UnstartedWorkflowState: types.ObjectUnknown(workflowStateAttrTypes),
		StartedWorkflowState:   types.ObjectUnknown(workflowStateAttrTypes),
		CompletedWorkflowState: types.ObjectUnknown(workflowStateAttrTypes),
		CanceledWorkflowState:  types.ObjectUnknown(workflowStateAttrTypes),
	}
	if diags := plan.Set(ctx, &data); diags.HasError() {
		t.Fatal(diags)
	}
	response := resource.CreateResponse{State: state}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	if len(client.requests) != 3 {
		t.Fatalf("unexpected requests: %#v", client.requests)
	}
	for _, request := range client.requests[:2] {
		encoded, err := json.Marshal(request.Variables)
		if err != nil {
			t.Fatal(err)
		}
		var variables struct {
			Input map[string]interface{} `json:"input"`
		}
		if err := json.Unmarshal(encoded, &variables); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"cyclesEnabled", "cycleDuration", "issueEstimationType", "defaultIssueEstimate"} {
			if _, exists := variables.Input[name]; exists {
				t.Errorf("%s sent %s for omitted setting: %s", request.OpName, name, encoded)
			}
		}
	}
}
