resource "linear_project_label" "release_tier" {
  name       = "Release Tier"
  group_type = "multiSelect"
  is_group   = true
}

resource "linear_project_label" "tier_1" {
  name      = "T1"
  parent_id = linear_project_label.release_tier.id
}

variable "team_id" {
  type = string
}

resource "linear_project_label" "team" {
  name    = "Team release"
  team_id = var.team_id
}
