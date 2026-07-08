resource functionApp 'Microsoft.Web/sites@2022-09-01' = {
  name: 'func-prod-001'
  location: 'eastus'
  kind: 'functionapp'
  properties: {
    serverFarmId: resourceId('Microsoft.Web/serverfarms', 'plan-prod-001')
    httpsOnly: true
  }
}

resource authSettings 'Microsoft.Web/sites/config@2022-09-01' = {
  parent: functionApp
  name: 'authsettingsV2'
  properties: {
    platform: {
      enabled: true
    }
    globalValidation: {
      requireAuthentication: false
      unauthenticatedClientAction: 'AllowAnonymous'
    }
    identityProviders: {
      azureActiveDirectory: {
        enabled: true
        registration: {
          openIdIssuer: 'https://sts.windows.net/00000000-0000-0000-0000-000000000000/'
          clientId: '11111111-1111-1111-1111-111111111111'
        }
      }
    }
  }
}
