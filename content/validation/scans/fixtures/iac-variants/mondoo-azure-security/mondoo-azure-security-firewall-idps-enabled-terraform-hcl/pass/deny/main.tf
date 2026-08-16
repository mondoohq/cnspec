resource "azurerm_firewall_policy" "example" {
  name                = "example-fw-policy"
  resource_group_name = "example-rg"
  location            = "eastus"

  intrusion_detection {
    mode = "Deny"
  }
}
