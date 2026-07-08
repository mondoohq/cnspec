resource "azuread_conditional_access_policy" "user_risk" {
  display_name = "Require MFA for medium and high user risk"
  state        = "enabled"

  conditions {
    client_app_types = ["all"]
    user_risk_levels = ["medium", "high"]

    applications {
      included_applications = ["All"]
    }

    users {
      included_users = ["All"]
    }
  }

  grant_controls {
    operator          = "OR"
    built_in_controls = ["mfa"]
  }
}
