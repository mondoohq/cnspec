resource site 'Microsoft.Web/sites@2023-12-01' = {
  name: 'contoso-webapp'
  location: 'eastus'
  identity: {
    type: 'UserAssigned'
    userAssignedIdentities: {
      '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-web/providers/Microsoft.ManagedIdentity/userAssignedIdentities/web-identity': {}
    }
  }
  properties: {
    httpsOnly: true
    siteConfig: {
      minTlsVersion: '1.2'
    }
  }
}
