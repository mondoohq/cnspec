resource aks 'Microsoft.ContainerService/managedClusters@2024-02-01' = {
  name: 'aksProdEastus'
  location: 'eastus'
  sku: {
    name: 'Base'
    tier: 'Standard'
  }
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    dnsPrefix: 'aksprod'
    kubernetesVersion: '1.28.5'
    enableRBAC: true
    agentPoolProfiles: [
      {
        name: 'systempool'
        count: 3
        vmSize: 'Standard_DS2_v2'
        mode: 'System'
        osType: 'Linux'
        type: 'VirtualMachineScaleSets'
      }
    ]
    apiServerAccessProfile: {
      authorizedIPRanges: [
        '203.0.113.0/24'
        '198.51.100.14/32'
      ]
      enablePrivateCluster: false
    }
  }
}
