package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type initiativeResourceTestClient struct {
	call func(*graphql.Request) (string, error)
}

func (c initiativeResourceTestClient) MakeRequest(_ context.Context, req *graphql.Request, resp *graphql.Response) error {
	payload, err := c.call(req)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(payload), resp.Data)
}

func TestInitiativeLabelLifecycle(t *testing.T) {
	ctx := context.Background()
	var input map[string]any
	var client graphql.Client = initiativeResourceTestClient{call: func(req *graphql.Request) (string, error) {
		vars, _ := json.Marshal(req.Variables)
		var decoded map[string]any
		_ = json.Unmarshal(vars, &decoded)
		if v, ok := decoded["input"]; ok {
			input = v.(map[string]any)
		}
		switch req.OpName {
		case "initiativeLabelCreate":
			return `{"initiativeLabelCreate":{"success":true,"initiativeLabel":{"id":"label-id","name":"Theme","description":null,"color":"#112233","isGroup":true,"groupType":"multiSelect","parent":null}}}`, nil
		case "initiativeLabelUpdate":
			return `{"initiativeLabelUpdate":{"success":true,"initiativeLabel":{"id":"label-id","name":"Theme","description":null,"color":"#112233","isGroup":true,"groupType":"singleSelect","parent":null}}}`, nil
		case "getInitiativeLabel":
			return "", errors.New("Entity not found: InitiativeLabel")
		case "initiativeLabelDelete":
			return `{"initiativeLabelDelete":{"success":false}}`, nil
		}
		return "", fmt.Errorf("unexpected operation %s", req.OpName)
	}}
	r := &InitiativeLabelResource{client: &client}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	model := InitiativeLabelResourceModel{Id: types.StringUnknown(), Name: types.StringValue("Theme"), Description: types.StringNull(), Color: types.StringUnknown(), IsGroup: types.BoolValue(true), GroupType: types.StringValue("multiSelect"), ParentId: types.StringNull()}
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
		t.Fatalf("group selection not sent: %#v", input)
	}
	if _, ok := input["color"]; ok {
		t.Fatalf("unknown color must use Linear's default: %#v", input)
	}
	if _, ok := input["retiredAt"]; ok {
		t.Fatal("ordinary create must not manage retirement")
	}
	if d := created.State.Get(ctx, &model); d.HasError() {
		t.Fatal(d)
	}
	if model.Id.ValueString() != "label-id" || model.Color.ValueString() != "#112233" {
		t.Fatalf("computed fields not reconciled: %#v", model)
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
	if input["groupType"] != "singleSelect" || input["parentId"] != nil || input["description"] != nil {
		t.Fatalf("update must support group selection and clearing optional fields: %#v", input)
	}
	deleted := resource.DeleteResponse{State: updated.State}
	r.Delete(ctx, resource.DeleteRequest{State: updated.State}, &deleted)
	if !deleted.Diagnostics.HasError() {
		t.Fatal("unsuccessful delete must retain state with a diagnostic")
	}
	read := resource.ReadResponse{State: updated.State}
	r.Read(ctx, resource.ReadRequest{State: updated.State}, &read)
	if read.Diagnostics.HasError() || !read.State.Raw.IsNull() {
		t.Fatalf("deleted label should be removed from state: %v", read.Diagnostics)
	}
}
