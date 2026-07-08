resource apim 'Microsoft.ApiManagement/service@2023-05-01-preview' = {
  name: 'apim-prod-003'
  location: 'eastus'
  sku: {
    name: 'Developer'
    capacity: 1
  }
  properties: {
    publisherEmail: 'apiteam@contoso.com'
    publisherName: 'Contoso'
    virtualNetworkType: 'None'
  }
}
