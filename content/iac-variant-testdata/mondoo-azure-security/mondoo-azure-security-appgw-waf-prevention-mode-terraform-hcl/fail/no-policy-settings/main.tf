resource "azurerm_web_application_firewall_policy" "fail" {
  name                = "example-wafpolicy"
  resource_group_name = "example-rg"
  location            = "eastus"

  managed_rules {
    managed_rule_set {
      type    = "OWASP"
      version = "3.2"
    }
  }
}
