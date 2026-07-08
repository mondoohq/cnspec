resource "azurerm_linux_web_app" "example" {
  name                = "example-web-app"
  resource_group_name = "example-rg"
  location            = "eastus"
  service_plan_id     = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.Web/serverfarms/example-plan"

  site_config {
    ip_restriction_default_action = "Deny"

    ip_restriction {
      ip_address = "10.0.0.0/24"
      action     = "Allow"
      priority   = 100
      name       = "allow-internal"
    }
  }
}
