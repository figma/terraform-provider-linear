package provider

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type initiativeLookupTestClient struct {
	call func(*graphql.Request) (string, error)
}

func (c initiativeLookupTestClient) MakeRequest(_ context.Context, req *graphql.Request, resp *graphql.Response) error {
	payload, err := c.call(req)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(payload), resp.Data)
}

func TestInitiativeLabelsPaginationAndAmbiguity(t *testing.T) {
	ctx := context.Background()
	calls := 0
	var client graphql.Client = initiativeLookupTestClient{call: func(req *graphql.Request) (string, error) {
		calls++
		raw, _ := json.Marshal(req.Variables)
		var vars map[string]any
		_ = json.Unmarshal(raw, &vars)
		if calls%2 == 1 {
			if vars["after"] != nil {
				t.Fatalf("unexpected initial cursor: %s", raw)
			}
			return `{"initiativeLabels":{"nodes":[{"id":"b","name":"Portfolio","color":"#112233","isGroup":false,"parent":{"id":"other"}}],"pageInfo":{"hasNextPage":true,"endCursor":"next"}}}`, nil
		}
		if vars["after"] != "next" {
			t.Fatalf("second page cursor missing: %s", raw)
		}
		return `{"initiativeLabels":{"nodes":[{"id":"a","name":"Portfolio","color":"#112233","isGroup":false,"parent":{"id":"wanted"}},{"id":"c","name":"Theme","color":"#112233","isGroup":true,"groupType":"singleSelect"}],"pageInfo":{"hasNextPage":false,"endCursor":null}}}`, nil
	}}
	labels, err := allInitiativeLabels(ctx, client, false)
	if err != nil || len(labels) != 3 || labels[0].Id != "a" {
		t.Fatalf("pagination/sorting failed: %#v %v", labels, err)
	}
	matches := filterInitiativeLabels(labels, types.StringValue("Portfolio"), types.StringValue("wanted"), types.BoolValue(false))
	if len(matches) != 1 || matches[0].Id != "a" {
		t.Fatalf("parent-scoped exact lookup failed: %#v", matches)
	}
	ds := &InitiativeLabelDataSource{client: &client}
	var schemaResp datasource.SchemaResponse
	ds.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)
	config := tfsdk.Config{Schema: schemaResp.Schema}
	state := tfsdk.State{Schema: schemaResp.Schema}
	model := InitiativeLabelLookupModel{Id: types.StringNull(), Name: types.StringValue("Portfolio"), Description: types.StringNull(), Color: types.StringNull(), IsGroup: types.BoolNull(), GroupType: types.StringNull(), ParentId: types.StringNull()}
	if d := state.Set(ctx, &model); d.HasError() {
		t.Fatal(d)
	}
	config.Raw = state.Raw
	response := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	ds.Read(ctx, datasource.ReadRequest{Config: config}, &response)
	if !response.Diagnostics.HasError() || !strings.Contains(response.Diagnostics.Errors()[0].Detail(), "found 2") {
		t.Fatalf("ambiguous lookup must fail: %v", response.Diagnostics)
	}
}
