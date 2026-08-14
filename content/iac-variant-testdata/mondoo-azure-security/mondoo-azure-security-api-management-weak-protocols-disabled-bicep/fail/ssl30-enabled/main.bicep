// SSL 3.0 is broken by POODLE; enabling it on the frontend downgrades every
// client that will negotiate it.
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
    customProperties: {
      'Microsoft.WindowsAzure.ApiManagement.Gateway.Security.Protocols.Ssl30': 'True'
      'Microsoft.WindowsAzure.ApiManagement.Gateway.Security.Backend.Protocols.Ssl30': 'False'
      'Microsoft.WindowsAzure.ApiManagement.Gateway.Security.Ciphers.TripleDes168': 'False'
    }
  }
}
