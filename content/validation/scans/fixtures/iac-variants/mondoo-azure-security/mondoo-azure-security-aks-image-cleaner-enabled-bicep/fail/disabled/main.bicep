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
    securityProfile: {
      imageCleaner: {
        enabled: false
      }
    }
  }
}
