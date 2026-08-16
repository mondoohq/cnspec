resource sbns 'Microsoft.ServiceBus/namespaces@2022-10-01-preview' = {
  name: 'contoso-sb-prod'
  location: 'eastus'
  sku: {
    name: 'Premium'
    tier: 'Premium'
  }
  properties: {
    minimumTlsVersion: '1.2'
    publicNetworkAccess: 'Disabled'
  }
}
