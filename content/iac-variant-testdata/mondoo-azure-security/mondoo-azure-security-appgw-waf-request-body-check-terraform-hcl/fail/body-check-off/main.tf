# With body inspection off the WAF only sees headers and the URL, so SQL
# injection and similar payloads delivered in a POST body pass straight through.
resource "azurerm_web_application_firewall_policy" "public" {
  name                = "public-waf"
  location            = azurerm_resource_group.prod.location
  resource_group_name = azurerm_resource_group.prod.name

  policy_settings {
    enabled            = true
    mode               = "Prevention"
    request_body_check = false
  }

  managed_rules {
    managed_rule_set {
      type    = "OWASP"
      version = "3.2"
    }
  }
}
