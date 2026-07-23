resource "azuread_conditional_access_policy" "user_risk" {
  display_name = "Require MFA or password change for user risk"
  state        = "enabled"

  conditions {
    client_app_types = ["all"]
    user_risk_levels = ["high"]

    applications {
      included_applications = ["All"]
    }

    users {
      included_users = ["All"]
    }
  }

  grant_controls {
    operator          = "AND"
    built_in_controls = ["mfa", "passwordChange"]
  }
}
