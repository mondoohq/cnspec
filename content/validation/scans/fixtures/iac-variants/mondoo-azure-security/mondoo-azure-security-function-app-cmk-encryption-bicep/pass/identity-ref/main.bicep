resource functionApp 'Microsoft.Web/sites@2023-12-01' = {
  name: 'func-prod-001'
  location: 'eastus'
  kind: 'functionapp'
  identity: {
    type: 'UserAssigned'
    userAssignedIdentities: {
      '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-prod/providers/Microsoft.ManagedIdentity/userAssignedIdentities/func-uami': {}
    }
  }
  properties: {
    serverFarmId: resourceId('Microsoft.Web/serverfarms', 'plan-prod-001')
    httpsOnly: true
    keyVaultReferenceIdentity: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-prod/providers/Microsoft.ManagedIdentity/userAssignedIdentities/func-uami'
  }
}
