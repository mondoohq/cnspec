resource "azuread_conditional_access_policy" "signin_risk" {
  display_name = "Require MFA for all users (no sign-in risk condition)"
  state        = "enabled"

  conditions {
    client_app_types = ["all"]

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
