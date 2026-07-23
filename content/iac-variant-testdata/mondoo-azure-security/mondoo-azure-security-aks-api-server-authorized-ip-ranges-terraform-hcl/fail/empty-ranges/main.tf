resource "azurerm_kubernetes_cluster" "fail" {
  name                = "example-aks"
  location            = "eastus"
  resource_group_name = "example-rg"
  dns_prefix          = "exampleaks"

  api_server_access_profile {
    authorized_ip_ranges = []
  }
}
