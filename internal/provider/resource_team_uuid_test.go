package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const testTeamUUID = "53c7964a-5bd4-4679-8cca-a5b78498b2b3"

type teamRecordingClient struct {
	requests  []*graphql.Request
	responses map[string]string
}

func (c *teamRecordingClient) MakeRequest(_ context.Context, request *graphql.Request, response *graphql.Response) error {
	c.requests = append(c.requests, request)
	payload, ok := c.responses[request.OpName]
	if !ok {
		return fmt.Errorf("unexpected GraphQL operation %q", request.OpName)
	}
	return json.Unmarshal([]byte(payload), response.Data)
}

func requestTeamID(t *testing.T, request *graphql.Request) string {
	t.Helper()
	encoded, err := json.Marshal(request.Variables)
	if err != nil {
		t.Fatalf("encode request variables: %v", err)
	}
	var variables struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(encoded, &variables); err != nil {
		t.Fatalf("decode request variables: %v", err)
	}
	return variables.ID
}

func teamResourceForUnitTest(t *testing.T, responses map[string]string) (*TeamResource, *teamRecordingClient, tfsdk.State) {
	t.Helper()
	var schemaResponse resource.SchemaResponse
	r := &TeamResource{}
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResponse)
	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("team schema: %v", schemaResponse.Diagnostics)
	}
	client := &teamRecordingClient{responses: responses}
	var graphQLClient graphql.Client = client
	r.client = &graphQLClient
	return r, client, tfsdk.State{
		Schema: schemaResponse.Schema,
		Raw:    tftypes.NewValue(schemaResponse.Schema.Type().TerraformType(context.Background()), nil),
	}
}

func TestTeamImportUsesUUIDAndAcceptsLegacyKey(t *testing.T) {
	for _, tc := range []struct {
		name       string
		identifier string
		attribute  string
	}{
		{"UUID", testTeamUUID, "id"},
		{"legacy key", "ENG", "key"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, _, state := teamResourceForUnitTest(t, nil)
			response := resource.ImportStateResponse{State: state}
			r.ImportState(context.Background(), resource.ImportStateRequest{ID: tc.identifier}, &response)
			if response.Diagnostics.HasError() {
				t.Fatalf("import diagnostics: %v", response.Diagnostics)
			}
			var got types.String
			response.Diagnostics.Append(response.State.GetAttribute(context.Background(), path.Root(tc.attribute), &got)...)
			if response.Diagnostics.HasError() || got.ValueString() != tc.identifier {
				t.Fatalf("imported %s = %q, diagnostics: %v", tc.attribute, got.ValueString(), response.Diagnostics)
			}
		})
	}
}

func TestTeamReadMissingUUIDFailsClosed(t *testing.T) {
	r, client, state := teamResourceForUnitTest(t, map[string]string{
		"getTeamByID": `{"administrableTeams":{"nodes":[]}}`,
	})
	if diags := state.SetAttribute(context.Background(), path.Root("id"), testTeamUUID); diags.HasError() {
		t.Fatalf("state ID: %v", diags)
	}
	if diags := state.SetAttribute(context.Background(), path.Root("key"), "ENG"); diags.HasError() {
		t.Fatalf("state key: %v", diags)
	}
	response := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &response)
	if !response.Diagnostics.HasError() {
		t.Fatal("missing UUID should stop refresh rather than discard the resource or find a reused key")
	}
	if len(client.requests) != 1 || client.requests[0].OpName != "getTeamByID" || requestTeamID(t, client.requests[0]) != testTeamUUID {
		t.Fatalf("unexpected lookup requests: %#v", client.requests)
	}
}

func TestTeamReadTracksRenamedKeyByUUID(t *testing.T) {
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
	r, client, state := teamResourceForUnitTest(t, map[string]string{
		"getTeamByID":           fmt.Sprintf(`{"administrableTeams":{"nodes":[{"id":%q,"key":"NEW","name":"Renamed team","timezone":"Etc/GMT","setIssueSortOrderOnStateChange":"first"}]}}`, testTeamUUID),
		"getTeamWorkflowStates": string(workflowPayload),
	})
	for attribute, value := range map[string]string{"id": testTeamUUID, "key": "OLD"} {
		if diags := state.SetAttribute(context.Background(), path.Root(attribute), value); diags.HasError() {
			t.Fatalf("state %s: %v", attribute, diags)
		}
	}
	response := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", response.Diagnostics)
	}
	var gotID, gotKey types.String
	response.Diagnostics.Append(response.State.GetAttribute(context.Background(), path.Root("id"), &gotID)...)
	response.Diagnostics.Append(response.State.GetAttribute(context.Background(), path.Root("key"), &gotKey)...)
	if response.Diagnostics.HasError() || gotID.ValueString() != testTeamUUID || gotKey.ValueString() != "NEW" {
		t.Fatalf("read returned ID %q, key %q, diagnostics: %v", gotID.ValueString(), gotKey.ValueString(), response.Diagnostics)
	}
	if len(client.requests) != 2 || client.requests[0].OpName != "getTeamByID" || client.requests[1].OpName != "getTeamWorkflowStates" || requestTeamID(t, client.requests[1]) != testTeamUUID {
		t.Fatalf("read must use UUID for team and workflows: %#v", client.requests)
	}
}

func TestTeamRemovalMakesOriginalUUIDPrivate(t *testing.T) {
	r, client, state := teamResourceForUnitTest(t, map[string]string{
		"makeTeamPrivate": fmt.Sprintf(`{"teamUpdate":{"team":{"id":%q,"private":true}}}`, testTeamUUID),
	})
	if diags := state.SetAttribute(context.Background(), path.Root("id"), testTeamUUID); diags.HasError() {
		t.Fatalf("state ID: %v", diags)
	}
	if diags := state.SetAttribute(context.Background(), path.Root("key"), "OLD"); diags.HasError() {
		t.Fatalf("state key: %v", diags)
	}
	response := resource.DeleteResponse{State: state}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %v", response.Diagnostics)
	}
	if len(client.requests) != 1 || client.requests[0].OpName != "makeTeamPrivate" || requestTeamID(t, client.requests[0]) != testTeamUUID {
		t.Fatalf("removal should target the original UUID: %#v", client.requests)
	}
	if !strings.Contains(client.requests[0].Query, "private:true") || strings.Contains(client.requests[0].Query, "parentId") || strings.Contains(client.requests[0].Query, "teamDelete") {
		t.Fatalf("removal should only make the team private: %s", client.requests[0].Query)
	}
}

func TestTeamRemovalRequiresPrivateConfirmation(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{"still public", fmt.Sprintf(`{"teamUpdate":{"team":{"id":%q,"private":false}}}`, testTeamUUID)},
		{"wrong team", `{"teamUpdate":{"team":{"id":"different-id","private":true}}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, _, state := teamResourceForUnitTest(t, map[string]string{"makeTeamPrivate": tc.body})
			if diags := state.SetAttribute(context.Background(), path.Root("id"), testTeamUUID); diags.HasError() {
				t.Fatalf("state ID: %v", diags)
			}
			response := resource.DeleteResponse{State: state}
			r.Delete(context.Background(), resource.DeleteRequest{State: state}, &response)
			if !response.Diagnostics.HasError() {
				t.Fatal("must not forget a team until Linear confirms its UUID and privacy")
			}
		})
	}
}
