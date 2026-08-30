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
      requireAuthentication: true
      unauthenticatedClientAction: 'RedirectToLoginPage'
    }
    identityProviders: {
      azureActiveDirectory: {
        enabled: true
      }
    }
  }
}
