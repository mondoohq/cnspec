resource "azurerm_linux_web_app" "pass" {
  name                       = "example-linux-app"
  location                   = "eastus"
  resource_group_name        = "example-rg"
  service_plan_id            = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.Web/serverfarms/example-plan"
  virtual_network_subnet_id  = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.Network/virtualNetworks/example-vnet/subnets/example-subnet"

  site_config {
    vnet_route_all_enabled = true
  }
}
