# Changelog

## Unreleased

### Enhancements
* Add workspace-level project and initiative labels, including label groups and group selection behavior.
* Add `linear_initiative_label` and `linear_initiative_labels` data sources for exact label lookups and filtered, paginated lists.

### Bug Fixes
* Preserve the existing `linear_project_label.group_type` when omitted from configuration so plans stay stable.

## 0.5.0

### Enhancements
* Track `linear_team` resources by UUID so key changes do not change resource identity. UUID and legacy key imports are supported.

### Behavior Changes
* Removing a `linear_team` makes that team private and removes it from Terraform state. Its issues, workflows, and parent remain in Linear; the provider no longer calls `teamDelete`.

## 0.4.1

### Security
* Update gRPC to 1.83.2, including upstream security fixes.

### Release
* Change the provider address to `spacelift.io/figma/linear`.

## 0.4.0

### Release
* Initial GitHub release of Figma's fork. This version was not published to a Terraform provider registry.

### Enhancements
* Added `linear_document` resource to manage Markdown documents attached to projects or issues, including import, reparenting, and moving documents to trash on deletion.
* Added `linear_project_label` resource with support for label groups, parent-child relationships, and import.
* Added `inherit_issue_estimation` and `inherit_workflow_statuses` to `linear_team` for sub-teams.

### Bug Fixes
* Read teams through `administrableTeams` so administrators can find private, restricted, retired, and archived teams.

## 0.3.7

### Bug Fixes
* Fix issue with created/updated labels being marked as archived

## 0.3.6

### Enhancements
* Added `is_group` to `linear_team_label` and `linear_workspace_label`

## 0.3.5

### Bug Fixes
* Allow `linear_team` to be created on free plan workspaces

## 0.3.4

### Enhancements
* Added `allow_members_to_manage_templates` to `linear_workspace_settings`
* Added `icon` and `color` to `linear_template`
* Added `allow_members_to_manage_team` to `linear_team`
* Added `allow_members_to_manage_members` to `linear_team`
* Added `allow_members_to_manage_labels` to `linear_team`
* Added `allow_members_to_manage_templates` to `linear_team`

## 0.3.3

### Enhancements
* Allow `linear_team` to be updated by non-admins

## 0.3.1

### Enhancements
* Added `linear_template` resource

## 0.3.0

### BREAKING
* Removed `no_priority_issues_first` from `linear_team`
* Move `enable_roadmap` to `initiatives.enabled` in `linear_workspace_settings`

### Enhancements
* Added `parent_id` to `linear_team` inorder to create sub-teams
* Added `enable_thread_summaries`, `auto_close_parent_issues`, `auto_close_child_issues` & `triage.require_priority` to `linear_team`
* Added `mergeable` to `linear_team_workflow`
* Added `branch` nested block to `linear_team_workflow`
* Added `allow_members_to_create_teams`, `allow_members_to_manage_labels` & `fiscal_year_start_month` to `linear_workspace_settings`
* Added `projects`, `initiatives`, `feed` & `customers` nested blocks to `linear_workspace_settings`
* Allow `triage` in `type` for `linear_workflow_state`

### Bug Fixes
* Fix issue with creating/updating team when cycles are enabled
* Fix issue with reading/creating/updating team workflow

## 0.2.6

### Bug Fixes
* Fix issue with creating/updating team caused by API change

## 0.2.5

### Enhancements
* Add `allow_members_to_invite` to workspace settings

### Bug Fixes
* Fix issues with API token
* Fix issues with updating workspace settings

## 0.2.3

### Enhancements
* Position of default workflow states is always `0`.

## 0.2.1

### Enhancements
* Improve recognition of existing workflows when importing a team

## 0.2.0 (First release)
