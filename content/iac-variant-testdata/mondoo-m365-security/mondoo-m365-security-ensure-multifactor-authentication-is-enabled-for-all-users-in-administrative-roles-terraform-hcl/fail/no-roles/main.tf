resource "azuread_conditional_access_policy" "admin_mfa" {
  display_name = "Require MFA for a named group instead of admin roles"
  state        = "enabled"

  conditions {
    client_app_types = ["all"]

    applications {
      included_applications = ["All"]
    }

    users {
      included_groups = [
        "11111111-2222-3333-4444-555555555555",
      ]
    }
  }

  grant_controls {
    operator          = "OR"
    built_in_controls = ["mfa"]
  }
}
