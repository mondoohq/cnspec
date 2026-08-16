resource "azuread_conditional_access_policy" "admin_mfa" {
  display_name = "Require MFA for administrator roles (staged)"
  state        = "disabled"

  conditions {
    client_app_types = ["all"]

    applications {
      included_applications = ["All"]
    }

    users {
      included_roles = [
        "62e90394-69f5-4237-9190-012177145e10",
      ]
    }
  }

  grant_controls {
    operator          = "OR"
    built_in_controls = ["mfa"]
  }
}
