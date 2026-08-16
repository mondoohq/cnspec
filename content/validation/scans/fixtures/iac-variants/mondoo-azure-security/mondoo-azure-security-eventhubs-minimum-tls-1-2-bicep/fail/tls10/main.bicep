resource ehns 'Microsoft.EventHub/namespaces@2024-01-01' = {
  name: 'ehns-prod-001'
  location: 'eastus'
  sku: {
    name: 'Standard'
    tier: 'Standard'
    capacity: 1
  }
  properties: {
    minimumTlsVersion: '1.0'
    disableLocalAuth: true
  }
}
