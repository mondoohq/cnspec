resource authConfig 'Microsoft.App/containerApps/authConfigs@2024-03-01' = {
  name: 'current'
  properties: {
    platform: {
      enabled: true
    }
    globalValidation: {
      unauthenticatedClientAction: 'RedirectToLoginPage'
    }
    identityProviders: {
      azureActiveDirectory: {
        enabled: true
      }
    }
  }
}
