resource "azurerm_role_definition" "example" {
  name        = "example-custom-role"
  scope       = "/subscriptions/00000000-0000-0000-0000-000000000000"
  description = "Custom role that can create role definitions"

  permissions {
    actions = [
      "Microsoft.Compute/virtualMachines/read",
      "Microsoft.Authorization/roleDefinitions/write",
    ]
    not_actions = []
  }

  assignable_scopes = [
    "/subscriptions/00000000-0000-0000-0000-000000000000",
  ]
}
