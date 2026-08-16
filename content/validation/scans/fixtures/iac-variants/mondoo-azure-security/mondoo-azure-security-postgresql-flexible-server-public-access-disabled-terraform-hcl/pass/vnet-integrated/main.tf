# Compliant: delegating a subnet puts the server in VNet-integrated mode, where
# Azure turns public network access off.
resource "azurerm_postgresql_flexible_server" "example" {
  name                = "example-psqlflexibleserver"
  resource_group_name = "example-rg"
  location            = "eastus"
  version             = "13"
  storage_mb          = 32768
  sku_name            = "GP_Standard_D4s_v3"

  administrator_login    = "psqladmin"
  administrator_password = "H@Sh1CoR3!"
  delegated_subnet_id    = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.Network/virtualNetworks/example-vnet/subnets/psql"
  private_dns_zone_id    = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.Network/privateDnsZones/example.postgres.database.azure.com"
}
