resource roleDef 'Microsoft.Authorization/roleDefinitions@2022-04-01' = {
  name: 'b24988ac-6180-42a0-ab88-20f7382dd24c'
  properties: {
    roleName: 'Custom Storage Reader'
    description: 'Read access to storage accounts and blobs'
    type: 'CustomRole'
    permissions: [
      {
        actions: [
          'Microsoft.Storage/storageAccounts/read'
          'Microsoft.Storage/storageAccounts/blobServices/containers/read'
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
