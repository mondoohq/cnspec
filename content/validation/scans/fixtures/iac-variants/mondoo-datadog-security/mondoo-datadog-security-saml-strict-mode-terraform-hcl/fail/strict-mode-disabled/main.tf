resource "datadog_organization_settings" "org" {
  name = "acme"

  settings {
    private_widget_share = false

    saml {
      enabled = true
    }

    saml_strict_mode {
      enabled = false
    }

    saml_idp_initiated_login {
      enabled = false
    }

    saml_autocreate_users_domains {
      enabled = true
      domains = ["acme.com"]
    }
  }
}
