resource endpoint 'Microsoft.MachineLearningServices/workspaces/onlineEndpoints@2023-10-01' = {
  name: 'mlworkspace/scoring-endpoint'
  location: 'eastus'
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    authMode: 'Key'
    description: 'Real-time scoring endpoint'
  }
}
