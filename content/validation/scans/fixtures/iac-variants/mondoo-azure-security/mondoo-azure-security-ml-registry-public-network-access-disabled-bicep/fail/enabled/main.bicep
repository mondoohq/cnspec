resource registry 'Microsoft.MachineLearningServices/registries@2024-10-01' = {
  name: 'example-registry'
  location: 'eastus'
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    publicNetworkAccess: 'Enabled'
  }
}
