resource registry 'Microsoft.MachineLearningServices/registries@2024-10-01' = {
  name: 'example-registry'
  location: 'eastus'
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    discoveryUrl: 'https://eastus.api.azureml.ms/registrymanagement/v1.0/registries/example-registry/discovery'
  }
}
