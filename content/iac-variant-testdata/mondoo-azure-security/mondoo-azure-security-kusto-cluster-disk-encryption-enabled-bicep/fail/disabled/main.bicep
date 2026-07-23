resource kustoCluster 'Microsoft.Kusto/clusters@2023-08-15' = {
  name: 'adxcontosodev'
  location: 'eastus'
  sku: {
    name: 'Standard_D13_v2'
    tier: 'Standard'
    capacity: 2
  }
  properties: {
    enableDiskEncryption: false
    enableStreamingIngest: true
    publicNetworkAccess: 'Enabled'
  }
}
