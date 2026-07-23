resource laCluster 'Microsoft.OperationalInsights/clusters@2022-10-01' = {
  name: 'la-cluster-contoso-prod'
  location: 'eastus'
  identity: {
    type: 'SystemAssigned'
  }
  sku: {
    name: 'CapacityReservation'
    capacity: 500
  }
  properties: {
    keyVaultProperties: {
      keyVaultUri: 'https://kv-la-prod.vault.azure.net'
      keyName: 'la-cmk'
      keyVersion: 'abcdef1234567890abcdef1234567890'
    }
  }
}
