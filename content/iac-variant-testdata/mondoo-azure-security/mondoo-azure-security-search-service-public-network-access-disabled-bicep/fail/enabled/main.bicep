resource search 'Microsoft.Search/searchServices@2023-11-01' = {
  name: 'srch-prod-eastus-001'
  location: 'eastus'
  sku: {
    name: 'standard'
  }
  properties: {
    replicaCount: 2
    partitionCount: 1
    hostingMode: 'default'
    publicNetworkAccess: 'Enabled'
  }
}
