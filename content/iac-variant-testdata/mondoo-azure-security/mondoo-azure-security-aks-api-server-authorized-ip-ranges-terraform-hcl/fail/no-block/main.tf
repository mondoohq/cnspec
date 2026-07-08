resource "azurerm_kubernetes_cluster" "fail" {
  name                = "example-aks"
  location            = "eastus"
  resource_group_name = "example-rg"
  dns_prefix          = "exampleaks"
}
