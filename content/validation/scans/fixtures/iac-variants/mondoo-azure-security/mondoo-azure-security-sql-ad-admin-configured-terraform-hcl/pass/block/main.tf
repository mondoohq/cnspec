resource "azurerm_mssql_server" "example" {
  name                         = "example-sqlserver"
  resource_group_name          = "example-rg"
  location                     = "eastus"
  version                      = "12.0"
  administrator_login          = "sqladmin"
  administrator_login_password = "P@ssw0rd1234!"

  azuread_administrator {
    login_username = "sqladmin_aad"
    object_id      = "00000000-0000-0000-0000-000000000000"
  }
}
