resource "linear_initiative_label" "portfolio" {
  name        = "Portfolio"
  description = "Coordinates projects around a shared release or rollout"
}

resource "linear_initiative_label" "theme" {
  name       = "Theme"
  is_group   = true
  group_type = "multiSelect"
}

resource "linear_initiative_label" "platform" {
  name      = "Platform"
  parent_id = linear_initiative_label.theme.id
}
