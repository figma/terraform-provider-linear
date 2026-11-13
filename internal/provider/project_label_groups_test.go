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

type projectGroupsTestClient struct {
	call func(*graphql.Request) (string, error)
}

func (c projectGroupsTestClient) MakeRequest(_ context.Context, req *graphql.Request, resp *graphql.Response) error {
	payload, err := c.call(req)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(payload), resp.Data)
}

func TestProjectLabelGroupSelection(t *testing.T) {
	ctx := context.Background()
	var input map[string]any
	var client graphql.Client = projectGroupsTestClient{call: func(req *graphql.Request) (string, error) {
		vars, _ := json.Marshal(req.Variables)
		var decoded map[string]any
		_ = json.Unmarshal(vars, &decoded)
		input = decoded["input"].(map[string]any)
		field := req.OpName
		return `{"` + field + `":{"success":true,"projectLabel":{"id":"label-id","name":"Release Tier","color":"#112233","isGroup":true,"groupType":"` + input["groupType"].(string) + `","team":{"id":"team-id"},"inheritedFrom":null}}}`, nil
	}}
	r := &ProjectLabelResource{client: &client}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	model := ProjectLabelResourceModel{Id: types.StringUnknown(), Name: types.StringValue("Release Tier"), Description: types.StringNull(), Color: types.StringUnknown(), IsGroup: types.BoolValue(true), GroupType: types.StringValue("multiSelect"), ParentId: types.StringNull()}
	plan := tfsdk.Plan{Schema: sr.Schema}
	if d := plan.Set(ctx, &model); d.HasError() {
		t.Fatal(d)
	}
	created := resource.CreateResponse{State: tfsdk.State{Schema: sr.Schema}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &created)
	if created.Diagnostics.HasError() {
		t.Fatal(created.Diagnostics)
	}
	if input["groupType"] != "multiSelect" {
		t.Fatalf("create must send group selection type: %#v", input)
	}
	if d := created.State.Get(ctx, &model); d.HasError() {
		t.Fatal(d)
	}
	model.GroupType = types.StringValue("singleSelect")
	if d := plan.Set(ctx, &model); d.HasError() {
		t.Fatal(d)
	}
	updated := resource.UpdateResponse{State: created.State}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: created.State}, &updated)
	if updated.Diagnostics.HasError() {
		t.Fatal(updated.Diagnostics)
	}

	if input["groupType"] != "singleSelect" {
		t.Fatal("update must send changed group selection")
	}
}
