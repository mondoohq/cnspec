resource compute 'Microsoft.MachineLearningServices/workspaces/computes@2023-10-01' = {
  name: 'mlworkspace/cpu-cluster'
  location: 'eastus'
  properties: {
    computeType: 'AmlCompute'
    properties: {
      vmSize: 'Standard_DS3_v2'
      enableNodePublicIp: false
      subnet: {
        id: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-ml/providers/Microsoft.Network/virtualNetworks/vnet/subnets/ml-subnet'
      }
      scaleSettings: {
        minNodeCount: 0
        maxNodeCount: 4
      }
    }
  }
}
