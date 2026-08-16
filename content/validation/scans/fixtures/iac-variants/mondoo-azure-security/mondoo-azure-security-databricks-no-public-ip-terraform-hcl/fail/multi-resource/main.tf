resource "azurerm_databricks_workspace" "compliant" {
  name                = "compliant-databricks"
  resource_group_name = "example-rg"
  location            = "eastus"
  sku                 = "premium"

  custom_parameters {
    no_public_ip        = true
    virtual_network_id  = azurerm_virtual_network.example.id
    public_subnet_name  = "public"
    private_subnet_name = "private"
  }
}

resource "azurerm_databricks_workspace" "violating" {
  name                = "violating-databricks"
  resource_group_name = "example-rg"
  location            = "eastus"
  sku                 = "premium"

  custom_parameters {
    no_public_ip = false
  }
}
