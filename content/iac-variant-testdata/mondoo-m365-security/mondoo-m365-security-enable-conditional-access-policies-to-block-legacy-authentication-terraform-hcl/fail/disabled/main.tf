resource "azuread_conditional_access_policy" "block_legacy_auth" {
  display_name = "Block legacy authentication (report-only)"
  state        = "disabled"

  conditions {
    client_app_types = ["exchangeActiveSync", "other"]

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
