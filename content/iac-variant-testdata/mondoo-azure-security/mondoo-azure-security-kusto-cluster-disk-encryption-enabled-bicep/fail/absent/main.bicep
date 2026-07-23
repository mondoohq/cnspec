resource kustoCluster 'Microsoft.Kusto/clusters@2023-08-15' = {
  name: 'adxcontosotest'
  location: 'eastus'
  sku: {
    name: 'Standard_D13_v2'
    tier: 'Standard'
    capacity: 2
  }
  properties: {
    enableStreamingIngest: true
  }
}
