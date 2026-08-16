resource "azuread_conditional_access_policy" "all_users_mfa" {
  display_name = "Require MFA for the pilot group only"
  state        = "enabled"

  conditions {
    client_app_types = ["all"]

    applications {
      included_applications = ["All"]
    }

    users {
      included_groups = [
        "22222222-3333-4444-5555-666666666666", # MFA pilot group
      ]
    }
  }

  grant_controls {
    operator          = "OR"
    built_in_controls = ["mfa"]
  }
}
