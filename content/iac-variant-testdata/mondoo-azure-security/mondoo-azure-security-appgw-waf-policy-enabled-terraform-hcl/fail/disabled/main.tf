# The policy exists and is attached, but is switched off, so no rule in it is
# evaluated. This reads as protected in an inventory while filtering nothing.
resource "azurerm_web_application_firewall_policy" "public" {
  name                = "public-waf"
  location            = azurerm_resource_group.prod.location
  resource_group_name = azurerm_resource_group.prod.name

  policy_settings {
    enabled = false
    mode    = "Detection"
  }

  managed_rules {
    managed_rule_set {
      type    = "OWASP"
      version = "3.2"
    }
  }
}
