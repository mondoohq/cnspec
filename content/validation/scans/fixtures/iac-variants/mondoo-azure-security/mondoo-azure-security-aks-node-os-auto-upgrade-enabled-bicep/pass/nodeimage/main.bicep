resource aks 'Microsoft.ContainerService/managedClusters@2024-02-01' = {
  name: 'production-aks'
  location: 'eastus'
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    dnsPrefix: 'prod-aks'
    kubernetesVersion: '1.29.2'
    enableRBAC: true
    agentPoolProfiles: [
      {
        name: 'systempool'
        count: 3
        vmSize: 'Standard_DS2_v2'
        mode: 'System'
        osType: 'Linux'
      }
    ]
    autoUpgradeProfile: {
      upgradeChannel: 'stable'
      nodeOSUpgradeChannel: 'NodeImage'
    }
  }
}
