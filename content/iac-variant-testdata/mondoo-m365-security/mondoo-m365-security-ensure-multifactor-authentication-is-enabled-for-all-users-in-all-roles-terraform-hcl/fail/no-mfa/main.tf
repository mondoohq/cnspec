resource "azuread_conditional_access_policy" "all_users_mfa" {
  display_name = "Require compliant device for all users (no MFA)"
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
    built_in_controls = ["compliantDevice"]
  }
}
