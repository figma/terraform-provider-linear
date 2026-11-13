package provider

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func randomAcceptanceTeamKey() (string, error) {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var random [5]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	key := make([]byte, len(random))
	key[0] = alphabet[int(random[0])%26]
	for i := 1; i < len(key); i++ {
		key[i] = alphabet[int(random[i])%len(alphabet)]
	}
	return string(key), nil
}

func testAccTeamKey(t *testing.T) string {
	t.Helper()
	key, err := randomAcceptanceTeamKey()
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func testAccCleanupTeams(s *terraform.State) error {
	client := graphql.NewClient("https://api.linear.app/graphql", &http.Client{
		Transport: &authedTransport{token: os.Getenv("LINEAR_TOKEN"), wrapped: http.DefaultTransport},
	})
	var teams []*terraform.ResourceState
	for _, team := range s.RootModule().Resources {
		if team.Type == "linear_team" {
			teams = append(teams, team)
		}
	}
	sort.Slice(teams, func(i, j int) bool {
		return teams[i].Primary.Attributes["parent_id"] != "" && teams[j].Primary.Attributes["parent_id"] == ""
	})
	for _, team := range teams {
		var result struct {
			TeamDelete struct {
				Success bool `json:"success"`
			} `json:"teamDelete"`
		}
		err := client.MakeRequest(context.Background(), &graphql.Request{
			OpName:    "cleanupAcceptanceTeam",
			Query:     `mutation cleanupAcceptanceTeam($id: String!) { teamDelete(id: $id) { success } }`,
			Variables: map[string]string{"id": team.Primary.ID},
		}, &graphql.Response{Data: &result})
		if err != nil {
			return fmt.Errorf("remove acceptance test team %s: %w", team.Primary.ID, err)
		}
		if !result.TeamDelete.Success {
			return fmt.Errorf("Linear did not remove acceptance test team %s", team.Primary.ID)
		}
	}
	return nil
}

func TestAccTeamResourceDefault(t *testing.T) {
	initialKey := testAccTeamKey(t)
	updatedKey := testAccTeamKey(t)
	for updatedKey == initialKey {
		updatedKey = testAccTeamKey(t)
	}
	initialName := "Acc Tests " + initialKey
	updatedName := "Acceptance " + initialKey
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCleanupTeams,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTeamResourceConfigDefault(initialKey, initialName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("linear_team.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "key", initialKey),
					resource.TestCheckResourceAttr("linear_team.test", "name", initialName),
					resource.TestCheckResourceAttr("linear_team.test", "private", "false"),
					resource.TestCheckNoResourceAttr("linear_team.test", "parent_id"),
					resource.TestCheckResourceAttr("linear_team.test", "inherit_issue_estimation", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "inherit_workflow_statuses", "false"),
					resource.TestCheckNoResourceAttr("linear_team.test", "description"),
					resource.TestCheckResourceAttr("linear_team.test", "icon", "Bank"),
					resource.TestMatchResourceAttr("linear_team.test", "color", colorRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "timezone", "Etc/GMT"),
					resource.TestCheckResourceAttr("linear_team.test", "enable_issue_history_grouping", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "enable_issue_default_to_bottom", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "enable_thread_summaries", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "auto_archive_period", "6"),
					resource.TestCheckResourceAttr("linear_team.test", "auto_close_period", "6"),
					resource.TestCheckResourceAttr("linear_team.test", "auto_close_parent_issues", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "auto_close_child_issues", "false"),
					resource.TestCheckNoResourceAttr("linear_team.test", "allow_members_to_manage_team"),
					resource.TestCheckNoResourceAttr("linear_team.test", "allow_members_to_manage_members"),
					resource.TestCheckNoResourceAttr("linear_team.test", "allow_members_to_manage_labels"),
					resource.TestCheckNoResourceAttr("linear_team.test", "allow_members_to_manage_templates"),
					resource.TestCheckResourceAttr("linear_team.test", "triage.enabled", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "triage.require_priority", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.enabled", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.start_day", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.duration", "1"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.cooldown", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.upcoming", "2"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.auto_add_started", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.auto_add_completed", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.need_for_active", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "estimation.type", "notUsed"),
					resource.TestCheckResourceAttr("linear_team.test", "estimation.extended", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "estimation.allow_zero", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "estimation.default", "1"),
					resource.TestMatchResourceAttr("linear_team.test", "backlog_workflow_state.id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "backlog_workflow_state.position", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "backlog_workflow_state.name", "Backlog"),
					resource.TestCheckResourceAttr("linear_team.test", "backlog_workflow_state.color", "#bec2c8"),
					resource.TestCheckNoResourceAttr("linear_team.test", "backlog_workflow_state.description"),
					resource.TestMatchResourceAttr("linear_team.test", "unstarted_workflow_state.id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "unstarted_workflow_state.position", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "unstarted_workflow_state.name", "Todo"),
					resource.TestCheckResourceAttr("linear_team.test", "unstarted_workflow_state.color", "#e2e2e2"),
					resource.TestCheckNoResourceAttr("linear_team.test", "unstarted_workflow_state.description"),
					resource.TestMatchResourceAttr("linear_team.test", "started_workflow_state.id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "started_workflow_state.position", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "started_workflow_state.name", "In Progress"),
					resource.TestCheckResourceAttr("linear_team.test", "started_workflow_state.color", "#f2c94c"),
					resource.TestCheckNoResourceAttr("linear_team.test", "started_workflow_state.description"),
					resource.TestMatchResourceAttr("linear_team.test", "completed_workflow_state.id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "completed_workflow_state.position", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "completed_workflow_state.name", "Done"),
					resource.TestCheckResourceAttr("linear_team.test", "completed_workflow_state.color", "#5e6ad2"),
					resource.TestCheckNoResourceAttr("linear_team.test", "completed_workflow_state.description"),
					resource.TestMatchResourceAttr("linear_team.test", "canceled_workflow_state.id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "canceled_workflow_state.position", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "canceled_workflow_state.name", "Canceled"),
					resource.TestCheckResourceAttr("linear_team.test", "canceled_workflow_state.color", "#95a2b3"),
					resource.TestCheckNoResourceAttr("linear_team.test", "canceled_workflow_state.description"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "linear_team.test",
				ImportState:       true,
				ImportStateId:     initialKey,
				ImportStateVerify: true,
			},
			// Update with default values
			{
				Config: testAccTeamResourceConfigDefault(initialKey, initialName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("linear_team.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "key", initialKey),
					resource.TestCheckResourceAttr("linear_team.test", "name", initialName),
					resource.TestCheckResourceAttr("linear_team.test", "private", "false"),
					resource.TestCheckNoResourceAttr("linear_team.test", "parent_id"),
					resource.TestCheckNoResourceAttr("linear_team.test", "description"),
					resource.TestCheckResourceAttr("linear_team.test", "icon", "Bank"),
					resource.TestMatchResourceAttr("linear_team.test", "color", colorRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "timezone", "Etc/GMT"),
					resource.TestCheckResourceAttr("linear_team.test", "enable_issue_history_grouping", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "enable_issue_default_to_bottom", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "enable_thread_summaries", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "auto_archive_period", "6"),
					resource.TestCheckResourceAttr("linear_team.test", "auto_close_period", "6"),
					resource.TestCheckResourceAttr("linear_team.test", "auto_close_parent_issues", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "auto_close_child_issues", "false"),
					resource.TestCheckNoResourceAttr("linear_team.test", "allow_members_to_manage_team"),
					resource.TestCheckNoResourceAttr("linear_team.test", "allow_members_to_manage_members"),
					resource.TestCheckNoResourceAttr("linear_team.test", "allow_members_to_manage_labels"),
					resource.TestCheckNoResourceAttr("linear_team.test", "allow_members_to_manage_templates"),
					resource.TestCheckResourceAttr("linear_team.test", "triage.enabled", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "triage.require_priority", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.enabled", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.start_day", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.duration", "1"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.cooldown", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.upcoming", "2"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.auto_add_started", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.auto_add_completed", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.need_for_active", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "estimation.type", "notUsed"),
					resource.TestCheckResourceAttr("linear_team.test", "estimation.extended", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "estimation.allow_zero", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "estimation.default", "1"),
					resource.TestMatchResourceAttr("linear_team.test", "backlog_workflow_state.id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "backlog_workflow_state.position", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "backlog_workflow_state.name", "Backlog"),
					resource.TestCheckResourceAttr("linear_team.test", "backlog_workflow_state.color", "#bec2c8"),
					resource.TestCheckNoResourceAttr("linear_team.test", "backlog_workflow_state.description"),
					resource.TestMatchResourceAttr("linear_team.test", "unstarted_workflow_state.id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "unstarted_workflow_state.position", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "unstarted_workflow_state.name", "Todo"),
					resource.TestCheckResourceAttr("linear_team.test", "unstarted_workflow_state.color", "#e2e2e2"),
					resource.TestCheckNoResourceAttr("linear_team.test", "unstarted_workflow_state.description"),
					resource.TestMatchResourceAttr("linear_team.test", "started_workflow_state.id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "started_workflow_state.position", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "started_workflow_state.name", "In Progress"),
					resource.TestCheckResourceAttr("linear_team.test", "started_workflow_state.color", "#f2c94c"),
					resource.TestCheckNoResourceAttr("linear_team.test", "started_workflow_state.description"),
					resource.TestMatchResourceAttr("linear_team.test", "completed_workflow_state.id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "completed_workflow_state.position", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "completed_workflow_state.name", "Done"),
					resource.TestCheckResourceAttr("linear_team.test", "completed_workflow_state.color", "#5e6ad2"),
					resource.TestCheckNoResourceAttr("linear_team.test", "completed_workflow_state.description"),
					resource.TestMatchResourceAttr("linear_team.test", "canceled_workflow_state.id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "canceled_workflow_state.position", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "canceled_workflow_state.name", "Canceled"),
					resource.TestCheckResourceAttr("linear_team.test", "canceled_workflow_state.color", "#95a2b3"),
					resource.TestCheckNoResourceAttr("linear_team.test", "canceled_workflow_state.description"),
				),
			},
			// Update and Read testing
			{
				Config: testAccTeamResourceConfigNonDefault(updatedKey, updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("linear_team.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "key", updatedKey),
					resource.TestCheckResourceAttr("linear_team.test", "name", updatedName),
					resource.TestCheckResourceAttr("linear_team.test", "private", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "parent_id", "ab69b823-f187-444d-8848-782045b86656"),
					resource.TestCheckResourceAttr("linear_team.test", "description", "nice team"),
					resource.TestCheckResourceAttr("linear_team.test", "icon", "Image"),
					resource.TestCheckResourceAttr("linear_team.test", "color", "#00ff00"),
					resource.TestCheckResourceAttr("linear_team.test", "timezone", "Europe/London"),
					resource.TestCheckResourceAttr("linear_team.test", "enable_issue_history_grouping", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "enable_issue_default_to_bottom", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "enable_thread_summaries", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "auto_archive_period", "3"),
					resource.TestCheckResourceAttr("linear_team.test", "auto_close_period", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "auto_close_parent_issues", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "auto_close_child_issues", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "allow_members_to_manage_team", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "allow_members_to_manage_members", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "allow_members_to_manage_labels", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "allow_members_to_manage_templates", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "triage.enabled", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "triage.require_priority", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.enabled", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.start_day", "6"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.duration", "3"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.cooldown", "1"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.upcoming", "4"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.auto_add_started", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.auto_add_completed", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.need_for_active", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "estimation.type", "linear"),
					resource.TestCheckResourceAttr("linear_team.test", "estimation.extended", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "estimation.allow_zero", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "estimation.default", "0"),
					resource.TestMatchResourceAttr("linear_team.test", "backlog_workflow_state.id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "backlog_workflow_state.position", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "backlog_workflow_state.name", "Icebox"),
					resource.TestCheckResourceAttr("linear_team.test", "backlog_workflow_state.color", "#bbbbbb"),
					resource.TestCheckResourceAttr("linear_team.test", "backlog_workflow_state.description", "Not planned"),
					resource.TestMatchResourceAttr("linear_team.test", "unstarted_workflow_state.id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "unstarted_workflow_state.position", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "unstarted_workflow_state.name", "Ready to start"),
					resource.TestCheckResourceAttr("linear_team.test", "unstarted_workflow_state.color", "#eeeeee"),
					resource.TestCheckResourceAttr("linear_team.test", "unstarted_workflow_state.description", "Planned"),
					resource.TestMatchResourceAttr("linear_team.test", "started_workflow_state.id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "started_workflow_state.position", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "started_workflow_state.name", "In flight"),
					resource.TestCheckResourceAttr("linear_team.test", "started_workflow_state.color", "#ffcccc"),
					resource.TestCheckResourceAttr("linear_team.test", "started_workflow_state.description", "Working on it"),
					resource.TestMatchResourceAttr("linear_team.test", "completed_workflow_state.id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "completed_workflow_state.position", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "completed_workflow_state.name", "Merged"),
					resource.TestCheckResourceAttr("linear_team.test", "completed_workflow_state.color", "#5566dd"),
					resource.TestCheckResourceAttr("linear_team.test", "completed_workflow_state.description", "Merged to main"),
					resource.TestMatchResourceAttr("linear_team.test", "canceled_workflow_state.id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "canceled_workflow_state.position", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "canceled_workflow_state.name", "Invalid"),
					resource.TestCheckResourceAttr("linear_team.test", "canceled_workflow_state.color", "#99aabb"),
					resource.TestCheckResourceAttr("linear_team.test", "canceled_workflow_state.description", "Not valid or not needed"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "linear_team.test",
				ImportState:       true,
				ImportStateId:     updatedKey,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestBoolPointerIfKnown(t *testing.T) {
	tests := []struct {
		name    string
		value   types.Bool
		wantNil bool
		want    bool
	}{
		{name: "null", value: types.BoolNull(), wantNil: true},
		{name: "unknown", value: types.BoolUnknown(), wantNil: true},
		{name: "false", value: types.BoolValue(false)},
		{name: "true", value: types.BoolValue(true), want: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := boolPointerIfKnown(test.value)
			if test.wantNil {
				if got != nil {
					t.Fatalf("boolPointerIfKnown() = %v, want nil", *got)
				}
				return
			}
			if got == nil || *got != test.want {
				t.Fatalf("boolPointerIfKnown() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestTeamResourceInheritancePlanValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		expectError string
	}{
		{
			name: "parent required",
			config: `
resource "linear_team" "test" {
  key                      = "TEST"
  name                     = "Test"
  inherit_issue_estimation = true
}`,
			expectError: "Missing Parent Team",
		},
		{
			name: "local estimation conflicts",
			config: `
resource "linear_team" "test" {
  key                      = "TEST"
  name                     = "Test"
  parent_id                = "ab69b823-f187-444d-8848-782045b86656"
  inherit_issue_estimation = true
  estimation = {
    type = "linear"
  }
}`,
			expectError: "Conflicting Estimation Settings",
		},
		{
			name: "local workflow conflicts",
			config: `
resource "linear_team" "test" {
  key                       = "TEST"
  name                      = "Test"
  parent_id                 = "ab69b823-f187-444d-8848-782045b86656"
  inherit_workflow_statuses = true
  started_workflow_state = {
    name = "Working"
  }
}`,
			expectError: "Conflicting Workflow Settings",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      "provider \"linear\" { token = \"test\" }\n" + test.config,
						PlanOnly:    true,
						ExpectError: regexp.MustCompile(test.expectError),
					},
				},
			})
		})
	}
}

func TestTeamResourceMixedInheritancePlans(t *testing.T) {
	configs := map[string]string{
		"estimation only": `
resource "linear_team" "test" {
  key                      = "TEST"
  name                     = "Test"
  parent_id                = "ab69b823-f187-444d-8848-782045b86656"
  inherit_issue_estimation = true
  started_workflow_state = {
    name = "Working"
  }
}`,
		"workflow only": `
resource "linear_team" "test" {
  key                       = "TEST"
  name                      = "Test"
  parent_id                 = "ab69b823-f187-444d-8848-782045b86656"
  inherit_workflow_statuses = true
  estimation = {
    type = "linear"
  }
}`,
	}

	for name, config := range configs {
		t.Run(name, func(t *testing.T) {
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:             "provider \"linear\" { token = \"test\" }\n" + config,
						PlanOnly:           true,
						ExpectNonEmptyPlan: true,
					},
				},
			})
		})
	}
}

func TestAccTeamResourceNonDefault(t *testing.T) {
	key := testAccTeamKey(t)
	name := "DevOps " + key
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCleanupTeams,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTeamResourceConfigNonDefault(key, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("linear_team.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "key", key),
					resource.TestCheckResourceAttr("linear_team.test", "name", name),
					resource.TestCheckResourceAttr("linear_team.test", "private", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "parent_id", "ab69b823-f187-444d-8848-782045b86656"),
					resource.TestCheckResourceAttr("linear_team.test", "description", "nice team"),
					resource.TestCheckResourceAttr("linear_team.test", "icon", "Image"),
					resource.TestCheckResourceAttr("linear_team.test", "color", "#00ff00"),
					resource.TestCheckResourceAttr("linear_team.test", "timezone", "Europe/London"),
					resource.TestCheckResourceAttr("linear_team.test", "enable_issue_history_grouping", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "enable_issue_default_to_bottom", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "enable_thread_summaries", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "auto_archive_period", "3"),
					resource.TestCheckResourceAttr("linear_team.test", "auto_close_period", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "auto_close_parent_issues", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "auto_close_child_issues", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "allow_members_to_manage_team", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "allow_members_to_manage_members", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "allow_members_to_manage_labels", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "allow_members_to_manage_templates", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "triage.enabled", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "triage.require_priority", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.enabled", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.start_day", "6"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.duration", "3"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.cooldown", "1"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.upcoming", "4"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.auto_add_started", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.auto_add_completed", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.need_for_active", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "estimation.type", "linear"),
					resource.TestCheckResourceAttr("linear_team.test", "estimation.extended", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "estimation.allow_zero", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "estimation.default", "0"),
					resource.TestMatchResourceAttr("linear_team.test", "backlog_workflow_state.id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "backlog_workflow_state.position", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "backlog_workflow_state.name", "Icebox"),
					resource.TestCheckResourceAttr("linear_team.test", "backlog_workflow_state.color", "#bbbbbb"),
					resource.TestCheckResourceAttr("linear_team.test", "backlog_workflow_state.description", "Not planned"),
					resource.TestMatchResourceAttr("linear_team.test", "unstarted_workflow_state.id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "unstarted_workflow_state.position", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "unstarted_workflow_state.name", "Ready to start"),
					resource.TestCheckResourceAttr("linear_team.test", "unstarted_workflow_state.color", "#eeeeee"),
					resource.TestCheckResourceAttr("linear_team.test", "unstarted_workflow_state.description", "Planned"),
					resource.TestMatchResourceAttr("linear_team.test", "started_workflow_state.id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "started_workflow_state.position", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "started_workflow_state.name", "In flight"),
					resource.TestCheckResourceAttr("linear_team.test", "started_workflow_state.color", "#ffcccc"),
					resource.TestCheckResourceAttr("linear_team.test", "started_workflow_state.description", "Working on it"),
					resource.TestMatchResourceAttr("linear_team.test", "completed_workflow_state.id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "completed_workflow_state.position", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "completed_workflow_state.name", "Merged"),
					resource.TestCheckResourceAttr("linear_team.test", "completed_workflow_state.color", "#5566dd"),
					resource.TestCheckResourceAttr("linear_team.test", "completed_workflow_state.description", "Merged to main"),
					resource.TestMatchResourceAttr("linear_team.test", "canceled_workflow_state.id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "canceled_workflow_state.position", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "canceled_workflow_state.name", "Invalid"),
					resource.TestCheckResourceAttr("linear_team.test", "canceled_workflow_state.color", "#99aabb"),
					resource.TestCheckResourceAttr("linear_team.test", "canceled_workflow_state.description", "Not valid or not needed"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "linear_team.test",
				ImportState:       true,
				ImportStateId:     key,
				ImportStateVerify: true,
			},
			// Update with same values
			{
				Config: testAccTeamResourceConfigNonDefault(key, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("linear_team.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "key", key),
					resource.TestCheckResourceAttr("linear_team.test", "name", name),
					resource.TestCheckResourceAttr("linear_team.test", "private", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "parent_id", "ab69b823-f187-444d-8848-782045b86656"),
					resource.TestCheckResourceAttr("linear_team.test", "description", "nice team"),
					resource.TestCheckResourceAttr("linear_team.test", "icon", "Image"),
					resource.TestCheckResourceAttr("linear_team.test", "color", "#00ff00"),
					resource.TestCheckResourceAttr("linear_team.test", "timezone", "Europe/London"),
					resource.TestCheckResourceAttr("linear_team.test", "enable_issue_history_grouping", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "enable_issue_default_to_bottom", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "enable_thread_summaries", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "auto_archive_period", "3"),
					resource.TestCheckResourceAttr("linear_team.test", "auto_close_period", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "auto_close_parent_issues", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "auto_close_child_issues", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "allow_members_to_manage_team", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "allow_members_to_manage_members", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "allow_members_to_manage_labels", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "allow_members_to_manage_templates", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "triage.enabled", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "triage.require_priority", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.enabled", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.start_day", "6"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.duration", "3"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.cooldown", "1"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.upcoming", "4"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.auto_add_started", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.auto_add_completed", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.need_for_active", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "estimation.type", "linear"),
					resource.TestCheckResourceAttr("linear_team.test", "estimation.extended", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "estimation.allow_zero", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "estimation.default", "0"),
					resource.TestMatchResourceAttr("linear_team.test", "backlog_workflow_state.id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "backlog_workflow_state.position", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "backlog_workflow_state.name", "Icebox"),
					resource.TestCheckResourceAttr("linear_team.test", "backlog_workflow_state.color", "#bbbbbb"),
					resource.TestCheckResourceAttr("linear_team.test", "backlog_workflow_state.description", "Not planned"),
					resource.TestMatchResourceAttr("linear_team.test", "unstarted_workflow_state.id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "unstarted_workflow_state.position", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "unstarted_workflow_state.name", "Ready to start"),
					resource.TestCheckResourceAttr("linear_team.test", "unstarted_workflow_state.color", "#eeeeee"),
					resource.TestCheckResourceAttr("linear_team.test", "unstarted_workflow_state.description", "Planned"),
					resource.TestMatchResourceAttr("linear_team.test", "started_workflow_state.id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "started_workflow_state.position", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "started_workflow_state.name", "In flight"),
					resource.TestCheckResourceAttr("linear_team.test", "started_workflow_state.color", "#ffcccc"),
					resource.TestCheckResourceAttr("linear_team.test", "started_workflow_state.description", "Working on it"),
					resource.TestMatchResourceAttr("linear_team.test", "completed_workflow_state.id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "completed_workflow_state.position", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "completed_workflow_state.name", "Merged"),
					resource.TestCheckResourceAttr("linear_team.test", "completed_workflow_state.color", "#5566dd"),
					resource.TestCheckResourceAttr("linear_team.test", "completed_workflow_state.description", "Merged to main"),
					resource.TestMatchResourceAttr("linear_team.test", "canceled_workflow_state.id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "canceled_workflow_state.position", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "canceled_workflow_state.name", "Invalid"),
					resource.TestCheckResourceAttr("linear_team.test", "canceled_workflow_state.color", "#99aabb"),
					resource.TestCheckResourceAttr("linear_team.test", "canceled_workflow_state.description", "Not valid or not needed"),
				),
			},
			// Update with null values
			{
				Config: testAccTeamResourceConfigDefault(key, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("linear_team.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "key", key),
					resource.TestCheckResourceAttr("linear_team.test", "name", name),
					resource.TestCheckResourceAttr("linear_team.test", "private", "false"),
					resource.TestCheckNoResourceAttr("linear_team.test", "parent_id"),
					resource.TestCheckNoResourceAttr("linear_team.test", "description"),
					resource.TestCheckResourceAttr("linear_team.test", "icon", "Image"),
					resource.TestCheckResourceAttr("linear_team.test", "color", "#00ff00"),
					resource.TestCheckResourceAttr("linear_team.test", "timezone", "Etc/GMT"),
					resource.TestCheckResourceAttr("linear_team.test", "enable_issue_history_grouping", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "enable_issue_default_to_bottom", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "enable_thread_summaries", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "auto_archive_period", "6"),
					resource.TestCheckResourceAttr("linear_team.test", "auto_archive_period", "6"),
					resource.TestCheckResourceAttr("linear_team.test", "auto_close_parent_issues", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "auto_close_child_issues", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "allow_members_to_manage_team", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "allow_members_to_manage_members", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "allow_members_to_manage_labels", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "allow_members_to_manage_templates", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "triage.enabled", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "triage.require_priority", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.enabled", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.start_day", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.duration", "1"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.cooldown", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.upcoming", "2"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.auto_add_started", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.auto_add_completed", "true"),
					resource.TestCheckResourceAttr("linear_team.test", "cycles.need_for_active", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "estimation.type", "notUsed"),
					resource.TestCheckResourceAttr("linear_team.test", "estimation.extended", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "estimation.allow_zero", "false"),
					resource.TestCheckResourceAttr("linear_team.test", "estimation.default", "1"),
					resource.TestMatchResourceAttr("linear_team.test", "backlog_workflow_state.id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "backlog_workflow_state.position", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "backlog_workflow_state.name", "Backlog"),
					resource.TestCheckResourceAttr("linear_team.test", "backlog_workflow_state.color", "#bec2c8"),
					resource.TestCheckNoResourceAttr("linear_team.test", "backlog_workflow_state.description"),
					resource.TestMatchResourceAttr("linear_team.test", "unstarted_workflow_state.id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "unstarted_workflow_state.position", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "unstarted_workflow_state.name", "Todo"),
					resource.TestCheckResourceAttr("linear_team.test", "unstarted_workflow_state.color", "#e2e2e2"),
					resource.TestCheckNoResourceAttr("linear_team.test", "unstarted_workflow_state.description"),
					resource.TestMatchResourceAttr("linear_team.test", "started_workflow_state.id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "started_workflow_state.position", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "started_workflow_state.name", "In Progress"),
					resource.TestCheckResourceAttr("linear_team.test", "started_workflow_state.color", "#f2c94c"),
					resource.TestCheckNoResourceAttr("linear_team.test", "started_workflow_state.description"),
					resource.TestMatchResourceAttr("linear_team.test", "completed_workflow_state.id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "completed_workflow_state.position", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "completed_workflow_state.name", "Done"),
					resource.TestCheckResourceAttr("linear_team.test", "completed_workflow_state.color", "#5e6ad2"),
					resource.TestCheckNoResourceAttr("linear_team.test", "completed_workflow_state.description"),
					resource.TestMatchResourceAttr("linear_team.test", "canceled_workflow_state.id", uuidRegex()),
					resource.TestCheckResourceAttr("linear_team.test", "canceled_workflow_state.position", "0"),
					resource.TestCheckResourceAttr("linear_team.test", "canceled_workflow_state.name", "Canceled"),
					resource.TestCheckResourceAttr("linear_team.test", "canceled_workflow_state.color", "#95a2b3"),
					resource.TestCheckNoResourceAttr("linear_team.test", "canceled_workflow_state.description"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "linear_team.test",
				ImportState:       true,
				ImportStateId:     key,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccTeamResourceInheritance(t *testing.T) {
	keys := [3]string{testAccTeamKey(t), testAccTeamKey(t), testAccTeamKey(t)}
	for keys[1] == keys[0] {
		keys[1] = testAccTeamKey(t)
	}
	for keys[2] == keys[0] || keys[2] == keys[1] {
		keys[2] = testAccTeamKey(t)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCleanupTeams,
		Steps: []resource.TestStep{
			{
				Config: testAccTeamResourceConfigInheritance(true, "parent", keys),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("linear_team.child", "parent_id", "linear_team.parent", "id"),
					resource.TestCheckResourceAttr("linear_team.child", "inherit_issue_estimation", "true"),
					resource.TestCheckResourceAttr("linear_team.child", "inherit_workflow_statuses", "true"),
					resource.TestCheckResourceAttr("linear_team.child", "estimation.type", "linear"),
					resource.TestCheckResourceAttr("linear_team.child", "estimation.extended", "true"),
					resource.TestCheckResourceAttr("linear_team.child", "estimation.allow_zero", "true"),
					resource.TestCheckResourceAttr("linear_team.child", "estimation.default", "0"),
					resource.TestCheckResourceAttr("linear_team.child", "backlog_workflow_state.name", "Icebox"),
					resource.TestCheckResourceAttr("linear_team.child", "unstarted_workflow_state.name", "Ready"),
					resource.TestCheckResourceAttr("linear_team.child", "started_workflow_state.name", "Building"),
					resource.TestCheckResourceAttr("linear_team.child", "completed_workflow_state.name", "Shipped"),
					resource.TestCheckResourceAttr("linear_team.child", "canceled_workflow_state.name", "Declined"),
				),
			},
			{
				Config: testAccTeamResourceConfigInheritance(true, "alternate_parent", keys),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("linear_team.child", "parent_id", "linear_team.alternate_parent", "id"),
					resource.TestCheckResourceAttr("linear_team.child", "estimation.type", "exponential"),
					resource.TestCheckResourceAttr("linear_team.child", "backlog_workflow_state.name", "Alternate icebox"),
					resource.TestCheckResourceAttr("linear_team.child", "unstarted_workflow_state.name", "Alternate ready"),
					resource.TestCheckResourceAttr("linear_team.child", "started_workflow_state.name", "Alternate building"),
					resource.TestCheckResourceAttr("linear_team.child", "completed_workflow_state.name", "Alternate shipped"),
					resource.TestCheckResourceAttr("linear_team.child", "canceled_workflow_state.name", "Alternate declined"),
				),
			},
			{
				Config: testAccTeamResourceConfigInheritance(false, "alternate_parent", keys),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("linear_team.child", "inherit_issue_estimation", "false"),
					resource.TestCheckResourceAttr("linear_team.child", "inherit_workflow_statuses", "false"),
					resource.TestCheckResourceAttr("linear_team.child", "estimation.type", "fibonacci"),
					resource.TestCheckResourceAttr("linear_team.child", "backlog_workflow_state.name", "Child backlog"),
					resource.TestCheckResourceAttr("linear_team.child", "unstarted_workflow_state.name", "Child ready"),
					resource.TestCheckResourceAttr("linear_team.child", "started_workflow_state.name", "Child building"),
					resource.TestCheckResourceAttr("linear_team.child", "completed_workflow_state.name", "Child shipped"),
					resource.TestCheckResourceAttr("linear_team.child", "canceled_workflow_state.name", "Child declined"),
				),
			},
			{
				Config: testAccTeamResourceConfigInheritance(true, "parent", keys),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("linear_team.child", "inherit_issue_estimation", "true"),
					resource.TestCheckResourceAttr("linear_team.child", "inherit_workflow_statuses", "true"),
					resource.TestCheckResourceAttr("linear_team.child", "estimation.type", "linear"),
					resource.TestCheckResourceAttr("linear_team.child", "backlog_workflow_state.name", "Icebox"),
					resource.TestCheckResourceAttr("linear_team.child", "unstarted_workflow_state.name", "Ready"),
					resource.TestCheckResourceAttr("linear_team.child", "started_workflow_state.name", "Building"),
					resource.TestCheckResourceAttr("linear_team.child", "completed_workflow_state.name", "Shipped"),
					resource.TestCheckResourceAttr("linear_team.child", "canceled_workflow_state.name", "Declined"),
				),
			},
			{
				ResourceName:      "linear_team.child",
				ImportState:       true,
				ImportStateId:     keys[2],
				ImportStateVerify: true,
			},
			{
				Config:   testAccTeamResourceConfigInheritance(true, "parent", keys),
				PlanOnly: true,
			},
		},
	})
}

func testAccTeamResourceConfigDefault(key string, name string) string {
	return fmt.Sprintf(`
resource "linear_team" "test" {
  key = "%s"
  name = "%s"
}
`, key, name)
}

func testAccTeamResourceConfigNonDefault(key string, name string) string {
	return fmt.Sprintf(`
resource "linear_team" "test" {
  key = "%s"
  name = "%s"
  private = true
  parent_id = "ab69b823-f187-444d-8848-782045b86656"
  inherit_workflow_statuses = false
  description = "nice team"
  icon = "Image"
  color = "#00ff00"
  timezone = "Europe/London"
  enable_issue_history_grouping = false
  enable_issue_default_to_bottom = true
  enable_thread_summaries = false
  auto_archive_period = 3
  auto_close_period = 0
  auto_close_parent_issues = true
  auto_close_child_issues = true
  allow_members_to_manage_team = false
  allow_members_to_manage_members = false
  allow_members_to_manage_labels = true
  allow_members_to_manage_templates = true

  triage = {
    enabled          = true
    require_priority = true
  }

  cycles = {
    enabled = true
    start_day = 6
    duration = 3
    cooldown = 1
    upcoming = 4
    auto_add_started = false
    auto_add_completed = false
    # need_for_active = true
  }

  estimation = {
    type = "linear"
    extended = true
    allow_zero = true
    default = 0
  }

  backlog_workflow_state = {
    name = "Icebox"
    color = "#bbbbbb"
    description = "Not planned"
  }

  unstarted_workflow_state = {
    name = "Ready to start"
    color = "#eeeeee"
    description = "Planned"
  }

  started_workflow_state = {
    name = "In flight"
    color = "#ffcccc"
    description = "Working on it"
  }

  completed_workflow_state = {
    name = "Merged"
    color = "#5566dd"
    description = "Merged to main"
  }

  canceled_workflow_state = {
    name = "Invalid"
    color = "#99aabb"
    description = "Not valid or not needed"
  }
}
`, key, name)
}

func testAccTeamResourceConfigInheritance(inherit bool, parentResource string, keys [3]string) string {
	childSettings := `
  inherit_issue_estimation  = true
  inherit_workflow_statuses = true`
	if !inherit {
		childSettings = `
  inherit_issue_estimation  = false
  inherit_workflow_statuses = false

  estimation = {
    type = "fibonacci"
  }

  backlog_workflow_state = {
    name = "Child backlog"
  }

  unstarted_workflow_state = {
    name = "Child ready"
  }

  started_workflow_state = {
    name = "Child building"
  }

  completed_workflow_state = {
    name = "Child shipped"
  }

  canceled_workflow_state = {
    name = "Child declined"
  }`
	}

	return fmt.Sprintf(`
resource "linear_team" "parent" {
  key  = "%s"
  name = "Acceptance Parent %s"

  estimation = {
    type       = "linear"
    extended   = true
    allow_zero = true
    default    = 0
  }

  backlog_workflow_state = {
    name = "Icebox"
  }

  unstarted_workflow_state = {
    name = "Ready"
  }

  started_workflow_state = {
    name = "Building"
  }

  completed_workflow_state = {
    name = "Shipped"
  }

  canceled_workflow_state = {
    name = "Declined"
  }
}

resource "linear_team" "alternate_parent" {
  key  = "%s"
  name = "Acceptance Alternate Parent %s"

  estimation = {
    type = "exponential"
  }

  backlog_workflow_state = {
    name = "Alternate icebox"
  }

  unstarted_workflow_state = {
    name = "Alternate ready"
  }

  started_workflow_state = {
    name = "Alternate building"
  }

  completed_workflow_state = {
    name = "Alternate shipped"
  }

  canceled_workflow_state = {
    name = "Alternate declined"
  }
}

resource "linear_team" "child" {
  key       = "%s"
  name      = "Acceptance Child %s"
  parent_id = linear_team.%s.id
%s
}
`, keys[0], keys[0], keys[1], keys[1], keys[2], keys[2], parentResource, childSettings)
}
