resource compute 'Microsoft.MachineLearningServices/workspaces/computes@2023-10-01' = {
  name: 'mlworkspace/cpu-cluster'
  location: 'eastus'
  properties: {
    computeType: 'AmlCompute'
    properties: {
      vmSize: 'Standard_DS3_v2'
      enableNodePublicIp: true
      scaleSettings: {
        minNodeCount: 0
        maxNodeCount: 4
      }
    }
  }
}
