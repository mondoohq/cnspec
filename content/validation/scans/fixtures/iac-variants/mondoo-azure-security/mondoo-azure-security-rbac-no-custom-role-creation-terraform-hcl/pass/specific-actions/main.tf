resource "azurerm_role_definition" "example" {
  name        = "example-custom-role"
  scope       = "/subscriptions/00000000-0000-0000-0000-000000000000"
  description = "Custom role that cannot create role definitions"

  permissions {
    actions = [
      "Microsoft.Compute/virtualMachines/read",
      "Microsoft.Authorization/roleAssignments/read",
      "Microsoft.Storage/storageAccounts/read",
    ]
    not_actions = []
  }

  assignable_scopes = [
    "/subscriptions/00000000-0000-0000-0000-000000000000",
  ]
}
