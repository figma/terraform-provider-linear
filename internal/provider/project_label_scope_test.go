package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type projectScopeTestClient struct {
	call func(*graphql.Request) (string, error)
}

func (c projectScopeTestClient) MakeRequest(_ context.Context, req *graphql.Request, resp *graphql.Response) error {
	payload, err := c.call(req)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(payload), resp.Data)
}

func TestProjectLabelTeamScope(t *testing.T) {
	ctx := context.Background()
	var input map[string]any
	var client graphql.Client = projectScopeTestClient{call: func(req *graphql.Request) (string, error) {
		if req.OpName == "getProjectLabel" {
			return `{"projectLabel":{"id":"copy-id","name":"Release Tier","color":"#112233","isGroup":true,"team":{"id":"child-team"},"inheritedFrom":{"id":"label-id"}}}`, nil
		}
		vars, _ := json.Marshal(req.Variables)
		var decoded map[string]any
		_ = json.Unmarshal(vars, &decoded)
		input = decoded["input"].(map[string]any)
		field := req.OpName
		return `{"` + field + `":{"success":true,"projectLabel":{"id":"label-id","name":"Release Tier","color":"#112233","isGroup":true,"groupType":"multiSelect","team":{"id":"team-id"},"inheritedFrom":null}}}`, nil
	}}
	r := &ProjectLabelResource{client: &client}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	model := ProjectLabelResourceModel{Id: types.StringUnknown(), Name: types.StringValue("Release Tier"), Description: types.StringNull(), Color: types.StringUnknown(), IsGroup: types.BoolValue(true), ParentId: types.StringNull(), TeamId: types.StringValue("team-id"), InheritedFromId: types.StringUnknown()}
	plan := tfsdk.Plan{Schema: sr.Schema}
	if d := plan.Set(ctx, &model); d.HasError() {
		t.Fatal(d)
	}
	created := resource.CreateResponse{State: tfsdk.State{Schema: sr.Schema}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &created)
	if created.Diagnostics.HasError() {
		t.Fatal(created.Diagnostics)
	}
	if input["teamId"] != "team-id" {
		t.Fatalf("create must send team scope: %#v", input)
	}
	if d := created.State.Get(ctx, &model); d.HasError() {
		t.Fatal(d)
	}
	if model.TeamId.ValueString() != "team-id" || !model.InheritedFromId.IsNull() {
		t.Fatalf("incorrect source label scope: %#v", model)
	}
	if d := plan.Set(ctx, &model); d.HasError() {
		t.Fatal(d)
	}
	updated := resource.UpdateResponse{State: created.State}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: created.State}, &updated)
	if updated.Diagnostics.HasError() {
		t.Fatal(updated.Diagnostics)
	}
	if _, ok := input["teamId"]; ok {
		t.Fatalf("scope must not be sent on update: %#v", input)
	}
	model.Id = types.StringValue("copy-id")
	inheritedState := tfsdk.State{Schema: sr.Schema}
	if d := inheritedState.Set(ctx, &model); d.HasError() {
		t.Fatal(d)
	}
	read := resource.ReadResponse{State: inheritedState}
	r.Read(ctx, resource.ReadRequest{State: inheritedState}, &read)
	if read.Diagnostics.HasError() {
		t.Fatal(read.Diagnostics)
	}
	if d := read.State.Get(ctx, &model); d.HasError() {
		t.Fatal(d)
	}
	if model.TeamId.ValueString() != "child-team" || model.InheritedFromId.ValueString() != "label-id" {
		t.Fatal("refresh must retain the inherited label's owning team and source label")
	}

}
