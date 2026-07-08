resource roleDef 'Microsoft.Authorization/roleDefinitions@2022-04-01' = {
  name: 'b24988ac-6180-42a0-ab88-20f7382dd24c'
  properties: {
    roleName: 'Custom Role Manager'
    description: 'Can create and manage custom role definitions'
    type: 'CustomRole'
    permissions: [
      {
        actions: [
          'Microsoft.Authorization/roleDefinitions/read'
          'Microsoft.Authorization/roleDefinitions/write'
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
