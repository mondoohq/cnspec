// No key vault properties, so the cluster's ingested data stays under the
// Microsoft-managed key.
resource kusto 'Microsoft.Kusto/clusters@2023-08-15' = {
  name: 'analytics'
  location: 'eastus'
  sku: {
    name: 'Standard_D13_v2'
    tier: 'Standard'
    capacity: 2
  }
  properties: {
    enableDiskEncryption: true
  }
}
