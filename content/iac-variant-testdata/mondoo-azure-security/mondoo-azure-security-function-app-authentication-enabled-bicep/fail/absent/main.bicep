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
      enabled: false
    }
  }
}
