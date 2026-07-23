resource laCluster 'Microsoft.OperationalInsights/clusters@2022-10-01' = {
  name: 'la-cluster-contoso-dev'
  location: 'eastus'
  identity: {
    type: 'SystemAssigned'
  }
  sku: {
    name: 'CapacityReservation'
    capacity: 500
  }
  properties: {
    billingType: 'Cluster'
  }
}
