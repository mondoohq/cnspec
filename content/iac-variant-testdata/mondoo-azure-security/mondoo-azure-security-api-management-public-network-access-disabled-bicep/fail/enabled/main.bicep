// The gateway and its management plane stay reachable from the public internet.
resource apim 'Microsoft.ApiManagement/service@2023-05-01-preview' = {
  name: 'contoso-apim'
  location: 'eastus'
  sku: {
    name: 'Developer'
    capacity: 1
  }
  properties: {
    publisherName: 'Example Corp'
    publisherEmail: 'platform@example.com'
    publicNetworkAccess: 'Enabled'
  }
}
