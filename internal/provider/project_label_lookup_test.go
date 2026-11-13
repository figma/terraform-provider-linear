package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type projectLookupTestClient struct {
	call func(*graphql.Request) (string, error)
}

func (c projectLookupTestClient) MakeRequest(_ context.Context, req *graphql.Request, resp *graphql.Response) error {
	payload, err := c.call(req)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(payload), resp.Data)
}

func TestProjectLabelLookupsPreserveTeamAndInheritance(t *testing.T) {
	ctx := context.Background()
	page := 0
	var client graphql.Client = projectLookupTestClient{call: func(req *graphql.Request) (string, error) {
		page++
		vars, _ := json.Marshal(req.Variables)
		var input map[string]any
		_ = json.Unmarshal(vars, &input)
		if input["includeArchived"] != true {
			t.Fatal("include_archived must be passed to every page")
		}
		if page == 1 {
			return `{"projectLabels":{"nodes":[{"id":"workspace-label","name":"T1","color":"#112233","isGroup":false,"team":null,"parent":{"id":"group"}}],"pageInfo":{"hasNextPage":true,"endCursor":"next"}}}`, nil
		}
		if input["after"] != "next" {
			t.Fatal("missing pagination cursor")
		}
		return `{"projectLabels":{"nodes":[{"id":"team-label","name":"T1","color":"#112233","isGroup":false,"team":{"id":"team-id"},"parent":{"id":"group"},"inheritedFrom":{"id":"source-label"}}],"pageInfo":{"hasNextPage":false,"endCursor":null}}}`, nil
	}}
	ds := &ProjectLabelsDataSource{ProjectLabelDataSource: ProjectLabelDataSource{client: &client}}
	var sr datasource.SchemaResponse
	ds.Schema(ctx, datasource.SchemaRequest{}, &sr)
	configModel := ProjectLabelsDataSourceModel{Id: types.StringNull(), Name: types.StringValue("T1"), TeamId: types.StringValue("team-id"), ParentId: types.StringValue("group"), IsGroup: types.BoolValue(false), IncludeArchived: types.BoolValue(true)}
	state := tfsdk.State{Schema: sr.Schema}
	if d := state.Set(ctx, &configModel); d.HasError() {
		t.Fatal(d)
	}
	resp := datasource.ReadResponse{State: tfsdk.State{Schema: sr.Schema}}
	ds.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: sr.Schema, Raw: state.Raw}}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	var got ProjectLabelsDataSourceModel
	if d := resp.State.Get(ctx, &got); d.HasError() {
		t.Fatal(d)
	}
	if len(got.Labels) != 1 || got.Labels[0].Id.ValueString() != "team-label" || got.Labels[0].InheritedFromId.ValueString() != "source-label" {
		t.Fatalf("team filter must preserve source metadata: %#v", got.Labels)
	}
}
