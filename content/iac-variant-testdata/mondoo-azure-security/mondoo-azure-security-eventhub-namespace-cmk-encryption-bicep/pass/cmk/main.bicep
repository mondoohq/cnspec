resource ehns 'Microsoft.EventHub/namespaces@2024-01-01' = {
  name: 'ehns-prod-001'
  location: 'eastus'
  sku: {
    name: 'Standard'
    tier: 'Standard'
    capacity: 1
  }
  identity: {
    type: 'UserAssigned'
    userAssignedIdentities: {
      '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-prod/providers/Microsoft.ManagedIdentity/userAssignedIdentities/id-eventhub': {}
    }
  }
  properties: {
    minimumTlsVersion: '1.2'
    disableLocalAuth: true
    encryption: {
      keySource: 'Microsoft.KeyVault'
      keyVaultProperties: [
        {
          keyName: 'eventhub-cmk'
          keyVaultUri: 'https://kv-prod-001.vault.azure.net'
          identity: {
            userAssignedIdentity: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-prod/providers/Microsoft.ManagedIdentity/userAssignedIdentities/id-eventhub'
          }
        }
      ]
    }
  }
}
