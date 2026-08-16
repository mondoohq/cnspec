# TLS 1.0 is deprecated and disallowed by PCI DSS; accepting it lets a client
# negotiate down to it.
resource "azurerm_mssql_managed_instance" "prod" {
  name                = "prod-sqlmi"
  resource_group_name = azurerm_resource_group.prod.name
  location            = azurerm_resource_group.prod.location

  administrator_login          = "sqladmin"
  administrator_login_password = var.sqlmi_password
  license_type                 = "BasePrice"
  subnet_id                    = azurerm_subnet.sqlmi.id
  sku_name                     = "GP_Gen5"
  vcores                       = 4
  storage_size_in_gb           = 32

  minimum_tls_version = "1.0"
}
