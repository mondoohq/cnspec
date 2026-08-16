resource sbNamespace 'Microsoft.ServiceBus/namespaces@2022-10-01-preview' = {
  name: 'sb-prod-eastus-001'
  location: 'eastus'
  sku: {
    name: 'Premium'
    tier: 'Premium'
  }
  properties: {
    minimumTlsVersion: '1.2'
    disableLocalAuth: true
  }
}
