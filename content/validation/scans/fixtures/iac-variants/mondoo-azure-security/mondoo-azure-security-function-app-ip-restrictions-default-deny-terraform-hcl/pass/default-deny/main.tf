resource "azurerm_linux_function_app" "processor" {
  name                = "event-processor"
  location            = azurerm_resource_group.prod.location
  resource_group_name = azurerm_resource_group.prod.name
  service_plan_id     = azurerm_service_plan.prod.id
  storage_account_name       = azurerm_storage_account.functions.name
  storage_account_access_key = azurerm_storage_account.functions.primary_access_key

  site_config {
    ip_restriction_default_action = "Deny"

    ip_restriction {
      name       = "corporate-egress"
      action     = "Allow"
      ip_address = "203.0.113.0/24"
      priority   = 100
    }
  }
}
