// Key auth is a shared static secret with no identity behind it, so calls
// cannot be attributed and the key cannot be scoped or conditionally accessed.
resource endpoint 'Microsoft.MachineLearningServices/workspaces/serverlessEndpoints@2024-04-01' = {
  name: 'scoring'
  location: 'eastus'
  properties: {
    authMode: 'Key'
    modelSettings: {
      modelId: 'azureml://registries/azureml/models/example/versions/1'
    }
  }
}
