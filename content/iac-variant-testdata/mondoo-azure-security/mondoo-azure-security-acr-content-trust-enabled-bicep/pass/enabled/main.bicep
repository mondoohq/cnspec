resource acr 'Microsoft.ContainerRegistry/registries@2023-07-01' = {
  name: 'contosoregistry'
  location: 'eastus'
  sku: {
    name: 'Premium'
  }
  properties: {
    adminUserEnabled: false
    policies: {
      trustPolicy: {
        type: 'Notary'
        status: 'enabled'
      }
    }
  }
}
