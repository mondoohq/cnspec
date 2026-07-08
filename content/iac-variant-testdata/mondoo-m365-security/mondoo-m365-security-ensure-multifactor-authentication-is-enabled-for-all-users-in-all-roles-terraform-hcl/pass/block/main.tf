resource "azuread_conditional_access_policy" "all_users_mfa" {
  display_name = "Require MFA for all users"
  state        = "enabled"

  conditions {
    client_app_types = ["all"]

    applications {
      included_applications = ["All"]
    }

    users {
      included_users = ["All"]
      excluded_users = [
        "00000000-0000-0000-0000-000000000000", # break-glass account
      ]
    }
  }

  grant_controls {
    operator          = "OR"
    built_in_controls = ["mfa"]
  }
}
