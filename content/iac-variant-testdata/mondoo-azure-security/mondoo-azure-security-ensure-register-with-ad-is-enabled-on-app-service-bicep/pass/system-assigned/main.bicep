resource site 'Microsoft.Web/sites@2023-12-01' = {
  name: 'contoso-webapp'
  location: 'eastus'
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    httpsOnly: true
    siteConfig: {
      minTlsVersion: '1.2'
    }
  }
}
