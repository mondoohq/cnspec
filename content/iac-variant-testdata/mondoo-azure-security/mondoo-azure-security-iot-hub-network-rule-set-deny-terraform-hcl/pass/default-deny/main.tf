resource "azurerm_iothub" "fleet" {
  name                = "fleet-hub"
  resource_group_name = azurerm_resource_group.prod.name
  location            = azurerm_resource_group.prod.location

  sku {
    name     = "S1"
    capacity = 1
  }

  network_rule_set {
    default_action                     = "Deny"
    apply_to_builtin_eventhub_endpoint = true

    ip_rule {
      name    = "corporate-egress"
      ip_mask = "203.0.113.0/24"
      action  = "Allow"
    }
  }
}
