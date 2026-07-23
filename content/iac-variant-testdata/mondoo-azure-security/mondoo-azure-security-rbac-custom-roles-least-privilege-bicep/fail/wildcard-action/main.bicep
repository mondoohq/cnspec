resource roleDef 'Microsoft.Authorization/roleDefinitions@2022-04-01' = {
  name: 'b24988ac-6180-42a0-ab88-20f7382dd24c'
  properties: {
    roleName: 'Custom Owner'
    description: 'Full access to all resources'
    type: 'CustomRole'
    permissions: [
      {
        actions: [
          '*'
        ]
        notActions: []
        dataActions: []
        notDataActions: []
      }
    ]
    assignableScopes: [
      '/subscriptions/00000000-0000-0000-0000-000000000000'
    ]
  }
}
