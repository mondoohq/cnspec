resource "azurerm_mysql_flexible_server" "srv" {
  name                   = "example-mysql-flex"
  resource_group_name    = "example-rg"
  location               = "eastus"
  administrator_login    = "adminuser"
  administrator_password = "H@Sh1CoR3!"
  sku_name               = "GP_Standard_D2ds_v4"
  delegated_subnet_id = "/subscriptions/000/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/vnet/subnets/mysqlsn"
  private_dns_zone_id = "/subscriptions/000/resourceGroups/rg/providers/Microsoft.Network/privateDnsZones/example.mysql.database.azure.com"
}
