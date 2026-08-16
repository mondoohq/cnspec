resource acr 'Microsoft.ContainerRegistry/registries@2023-07-01' = {
  name: 'contosoregistry'
  location: 'eastus'
  sku: {
    name: 'Basic'
  }
  properties: {
    adminUserEnabled: true
  }
}
