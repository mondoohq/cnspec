resource cluster 'Microsoft.MachineLearningServices/workspaces/computes@2024-10-01' = {
  name: 'mlworkspace/cpu-cluster'
  location: 'eastus'
  properties: {
    computeType: 'AmlCompute'
    disableLocalAuth: true
    properties: {
      vmSize: 'Standard_DS3_v2'
    }
  }
}

resource attached 'Microsoft.MachineLearningServices/workspaces/computes@2024-10-01' = {
  name: 'mlworkspace/aks-attached'
  location: 'eastus'
  properties: {
    computeType: 'Kubernetes'
    resourceId: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.ContainerService/managedClusters/aks'
  }
}
