resource compute 'Microsoft.MachineLearningServices/workspaces/computes@2024-10-01' = {
  name: 'mlworkspace/example'
  location: 'eastus'
  properties: {
    computeType: 'AmlCompute'
    disableLocalAuth: true
    properties: {
      vmSize: 'Standard_DS3_v2'
    }
  }
}
