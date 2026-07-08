resource apim 'Microsoft.ApiManagement/service@2023-05-01-preview' = {
  name: 'contoso-apim'
  location: 'eastus'
  sku: {
    name: 'Standard'
    capacity: 1
  }
  properties: {
    publisherEmail: 'apiteam@contoso.com'
    publisherName: 'Contoso'
    customProperties: {
      'Microsoft.WindowsAzure.ApiManagement.Gateway.Security.Protocols.Tls10': 'True'
      'Microsoft.WindowsAzure.ApiManagement.Gateway.Security.Protocols.Tls11': 'False'
      'Microsoft.WindowsAzure.ApiManagement.Gateway.Security.Backend.Protocols.Tls10': 'False'
      'Microsoft.WindowsAzure.ApiManagement.Gateway.Security.Backend.Protocols.Tls11': 'False'
    }
  }
}
