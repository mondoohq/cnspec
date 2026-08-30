// Authentication is configured but anonymous requests are still served, so the
// app's own code is the only thing between the internet and its data.
resource site 'Microsoft.Web/sites@2023-01-01' = {
  name: 'example-webapp'
  location: 'eastus'
  properties: {
    httpsOnly: true
  }
}

resource authSettings 'Microsoft.Web/sites/config@2023-01-01' = {
  parent: site
  name: 'authsettingsV2'
  properties: {
    platform: {
      enabled: true
    }
    globalValidation: {
      requireAuthentication: false
      unauthenticatedClientAction: 'AllowAnonymous'
    }
  }
}
