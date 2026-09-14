resource "datadog_organization_settings" "org" {
  name = "acme"

  settings {
    saml {
      enabled = true
    }

    saml_strict_mode {
      enabled = true
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
