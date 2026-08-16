# With no schedule block the active assignment is permanent, which turns PIM
# into ordinary standing access.
resource "azurerm_pim_active_role_assignment" "contributor" {
  scope              = data.azurerm_subscription.prod.id
  role_definition_id = "${data.azurerm_subscription.prod.id}${data.azurerm_role_definition.contributor.id}"
  principal_id       = var.platform_group_object_id
  justification      = "Platform team standing access"
}
