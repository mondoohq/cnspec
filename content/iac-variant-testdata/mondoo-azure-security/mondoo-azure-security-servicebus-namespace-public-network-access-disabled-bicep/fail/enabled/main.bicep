resource sbns 'Microsoft.ServiceBus/namespaces@2022-10-01-preview' = {
  name: 'contoso-sb-public'
  location: 'eastus'
  sku: {
    name: 'Standard'
    tier: 'Standard'
  }
  properties: {
    minimumTlsVersion: '1.2'
    publicNetworkAccess: 'Enabled'
  }
}
