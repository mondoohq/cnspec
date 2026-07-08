resource acr 'Microsoft.ContainerRegistry/registries@2023-07-01' = {
  name: 'contosoregistry'
  location: 'eastus'
  sku: {
    name: 'Premium'
  }
  identity: {
    type: 'UserAssigned'
    userAssignedIdentities: {
      '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-acr/providers/Microsoft.ManagedIdentity/userAssignedIdentities/acr-identity': {}
    }
  }
  properties: {
    adminUserEnabled: false
    encryption: {
      status: 'enabled'
      keyVaultProperties: {
        keyIdentifier: 'https://contoso-kv.vault.azure.net/keys/acr-cmk'
        identity: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-acr/providers/Microsoft.ManagedIdentity/userAssignedIdentities/acr-identity'
      }
    }
  }
}
