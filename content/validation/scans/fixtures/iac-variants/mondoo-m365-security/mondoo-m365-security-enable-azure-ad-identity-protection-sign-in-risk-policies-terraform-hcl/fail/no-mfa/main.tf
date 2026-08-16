resource "azuread_conditional_access_policy" "signin_risk" {
  display_name = "Block on sign-in risk without MFA"
  state        = "enabled"

  conditions {
    client_app_types    = ["all"]
    sign_in_risk_levels = ["high"]

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
