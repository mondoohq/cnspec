resource "azuread_conditional_access_policy" "block_legacy_auth" {
  display_name = "Block policy without client app type scoping"
  state        = "enabled"

  conditions {
    applications {
      included_applications = ["All"]
    }

    users {
      included_users = ["All"]
    }
  }

  grant_controls {
    operator          = "OR"
    built_in_controls = ["block"]
  }
}
