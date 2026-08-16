resource "azurerm_logic_app_workflow" "example" {
  name                = "example-workflow"
  location            = "eastus"
  resource_group_name = "example-rg"

  access_control {
    content {
      allowed_caller_ip_address_range = [
        "10.0.0.0/8",
      ]
    }
  }
}
