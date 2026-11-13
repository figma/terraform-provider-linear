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

func TestProjectLabelLookupsScopedPagination(t *testing.T) {
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
	for _, tc := range []struct {
		name      string
		team      types.String
		workspace types.Bool
		want      string
		source    types.String
	}{
		{"team", types.StringValue("team-id"), types.BoolNull(), "team-label", types.StringValue("source-label")},
		{"workspace", types.StringNull(), types.BoolValue(true), "workspace-label", types.StringNull()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page = 0
			configModel := ProjectLabelsDataSourceModel{Id: types.StringNull(), Name: types.StringValue("T1"), TeamId: tc.team, WorkspaceOnly: tc.workspace, ParentId: types.StringValue("group"), IsGroup: types.BoolValue(false), IncludeArchived: types.BoolValue(true)}
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
			if len(got.Labels) != 1 || got.Labels[0].Id.ValueString() != tc.want || !got.Labels[0].InheritedFromId.Equal(tc.source) {
				t.Fatalf("scope filtering must preserve inheritance metadata across pages: %#v", got.Labels)
			}
		})
	}

}

func TestProjectLabelWorkspaceLookup(t *testing.T) {
	ctx := context.Background()
	workspace := `{"id":"workspace-label","name":"T1","color":"#112233","isGroup":false,"team":null}`
	inherited := `{"id":"team-label","name":"T1","color":"#112233","isGroup":false,"team":{"id":"team-id"},"inheritedFrom":{"id":"workspace-label"}}`
	var client graphql.Client = projectLookupTestClient{call: func(req *graphql.Request) (string, error) {
		if req.OpName == "lookupProjectLabel" {
			return `{"projectLabel":` + inherited + `}`, nil
		}
		return `{"projectLabels":{"nodes":[` + inherited + `,` + workspace + `],"pageInfo":{"hasNextPage":false,"endCursor":null}}}`, nil
	}}
	ds := &ProjectLabelDataSource{client: &client}
	var sr datasource.SchemaResponse
	ds.Schema(ctx, datasource.SchemaRequest{}, &sr)
	for _, tc := range []struct {
		name      string
		id        types.String
		team      types.String
		workspace types.Bool
		want      string
	}{
		{"workspace name", types.StringNull(), types.StringNull(), types.BoolValue(true), "workspace-label"},
		{"team name", types.StringNull(), types.StringValue("team-id"), types.BoolNull(), "team-label"},
		{"unrestricted name remains ambiguous", types.StringNull(), types.StringNull(), types.BoolNull(), ""},
		{"workspace scope rejects team UUID", types.StringValue("team-label"), types.StringNull(), types.BoolValue(true), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			model := ProjectLabelDataSourceModel{Id: tc.id, Name: types.StringValue("T1"), TeamId: tc.team, WorkspaceOnly: tc.workspace}
			config := tfsdk.State{Schema: sr.Schema}
			if d := config.Set(ctx, &model); d.HasError() {
				t.Fatal(d)
			}
			resp := datasource.ReadResponse{State: tfsdk.State{Schema: sr.Schema}}
			ds.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: sr.Schema, Raw: config.Raw}}, &resp)
			if tc.want == "" {
				if !resp.Diagnostics.HasError() {
					t.Fatal("expected ambiguous or out-of-scope label error")
				}
				return
			}
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			var got ProjectLabelDataSourceModel
			if d := resp.State.Get(ctx, &got); d.HasError() {
				t.Fatal(d)
			}
			if got.Id.ValueString() != tc.want {
				t.Fatalf("got %s, want %s", got.Id, tc.want)
			}
		})
	}
}
