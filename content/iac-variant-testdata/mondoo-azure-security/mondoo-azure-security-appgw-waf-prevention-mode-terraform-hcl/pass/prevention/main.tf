resource "azurerm_web_application_firewall_policy" "pass" {
  name                = "example-wafpolicy"
  resource_group_name = "example-rg"
  location            = "eastus"

  policy_settings {
    enabled = true
    mode    = "Prevention"
  }

  managed_rules {
    managed_rule_set {
      type    = "OWASP"
      version = "3.2"
    }
  }
}
