resource "azurerm_mysql_flexible_server" "srv" {
  name                   = "example-mysql-flex"
  resource_group_name    = "example-rg"
  location               = "eastus"
  administrator_login    = "adminuser"
  administrator_password = "H@Sh1CoR3!"
  sku_name               = "GP_Standard_D2ds_v4"

  identity {
    type         = "UserAssigned"
    identity_ids = ["/subscriptions/000/resourceGroups/rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/uai"]
  }

  customer_managed_key {
    key_vault_key_id                  = "https://myvault.vault.azure.net/keys/mykey/version"
    primary_user_assigned_identity_id = "/subscriptions/000/resourceGroups/rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/uai"
  }
}
