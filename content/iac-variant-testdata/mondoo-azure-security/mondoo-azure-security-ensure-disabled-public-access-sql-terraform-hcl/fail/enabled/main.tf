resource "azurerm_mssql_server" "example" {
  name                          = "example-sqlserver"
  resource_group_name           = "example-rg"
  location                      = "eastus"
  version                       = "12.0"
  administrator_login           = "sqladmin"
  administrator_login_password  = "P@ssw0rd1234!"
  public_network_access_enabled = true
}
