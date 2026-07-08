resource site 'Microsoft.Web/sites@2022-09-01' = {
  name: 'app-prod-001'
  location: 'eastus'
  properties: {
    serverFarmId: resourceId('Microsoft.Web/serverfarms', 'plan-prod-001')
    httpsOnly: true
    publicNetworkAccess: 'Disabled'
  }
}
