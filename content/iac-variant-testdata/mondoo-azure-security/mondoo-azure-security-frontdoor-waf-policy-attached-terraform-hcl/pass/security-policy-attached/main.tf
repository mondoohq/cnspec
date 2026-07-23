resource "azurerm_cdn_frontdoor_profile" "example" {
  name                = "example-profile"
  resource_group_name = "example-rg"
  sku_name            = "Premium_AzureFrontDoor"
}

resource "azurerm_cdn_frontdoor_firewall_policy" "example" {
  name                = "exampleWAF"
  resource_group_name = "example-rg"
  sku_name            = "Premium_AzureFrontDoor"
  enabled             = true
  mode                = "Prevention"
}

resource "azurerm_cdn_frontdoor_security_policy" "example" {
  name                     = "example-security-policy"
  cdn_frontdoor_profile_id = azurerm_cdn_frontdoor_profile.example.id

  security_policies {
    firewall {
      cdn_frontdoor_firewall_policy_id = azurerm_cdn_frontdoor_firewall_policy.example.id

      association {
        domain {
          cdn_frontdoor_domain_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.Cdn/profiles/example-profile/customDomains/example-domain"
        }
        patterns_to_match = ["/*"]
      }
    }
  }
}
