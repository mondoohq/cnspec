resource "azurerm_role_definition" "example" {
  name        = "example-custom-role"
  scope       = "/subscriptions/00000000-0000-0000-0000-000000000000"
  description = "Custom role with least-privilege actions"

  permissions {
    actions = [
      "Microsoft.Compute/virtualMachines/read",
      "Microsoft.Compute/virtualMachines/start/action",
      "Microsoft.Storage/storageAccounts/read",
    ]
    not_actions = []
  }

  assignable_scopes = [
    "/subscriptions/00000000-0000-0000-0000-000000000000",
  ]
}
