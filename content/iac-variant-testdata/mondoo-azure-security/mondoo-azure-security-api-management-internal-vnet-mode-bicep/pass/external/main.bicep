resource apim 'Microsoft.ApiManagement/service@2023-05-01-preview' = {
  name: 'apim-prod-002'
  location: 'eastus'
  sku: {
    name: 'Premium'
    capacity: 2
  }
  properties: {
    publisherEmail: 'apiteam@contoso.com'
    publisherName: 'Contoso'
    virtualNetworkType: 'External'
  }
}
