resource workspace 'Microsoft.MachineLearningServices/workspaces@2024-10-01' = {
  name: 'example-workspace'
  location: 'eastus'
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    managedNetwork: {
      isolationMode: 'AllowInternetOutbound'
    }
  }
}
