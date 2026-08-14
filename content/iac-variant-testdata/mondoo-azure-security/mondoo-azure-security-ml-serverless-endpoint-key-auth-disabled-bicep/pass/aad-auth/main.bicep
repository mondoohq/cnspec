resource endpoint 'Microsoft.MachineLearningServices/workspaces/serverlessEndpoints@2024-04-01' = {
  name: 'scoring'
  location: 'eastus'
  properties: {
    authMode: 'AAD'
    modelSettings: {
      modelId: 'azureml://registries/azureml/models/example/versions/1'
    }
  }
}
