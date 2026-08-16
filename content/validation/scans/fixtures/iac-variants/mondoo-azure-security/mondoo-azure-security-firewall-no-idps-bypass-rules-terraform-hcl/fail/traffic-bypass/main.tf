# A bypass rule exempts traffic from intrusion detection entirely. Broad
# bypasses are how IDPS ends up enabled but blind on the paths that matter.
resource "azurerm_firewall_policy" "prod" {
  name                = "prod-policy"
  resource_group_name = azurerm_resource_group.prod.name
  location            = azurerm_resource_group.prod.location
  sku                 = "Premium"

  intrusion_detection {
    mode = "Deny"

    traffic_bypass {
      name              = "skip-partner-traffic"
      protocol          = "TCP"
      source_addresses  = ["10.0.0.0/8"]
      destination_ports = ["443"]
    }
  }
}
