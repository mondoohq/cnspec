resource "azurerm_pim_active_role_assignment" "contributor" {
  scope              = data.azurerm_subscription.prod.id
  role_definition_id = "${data.azurerm_subscription.prod.id}${data.azurerm_role_definition.contributor.id}"
  principal_id       = var.platform_group_object_id

  schedule {
    start_date_time = "2026-01-01T00:00:00Z"

    expiration {
      duration_hours = 8
    }
  }

  justification = "Scheduled maintenance window"
}
