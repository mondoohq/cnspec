resource "azurerm_snapshot" "example" {
  name                          = "snap-example"
  location                      = "eastus"
  resource_group_name           = "example-rg"
  create_option                 = "Copy"
  source_uri                    = azurerm_managed_disk.example.id
  public_network_access_enabled = true
}
