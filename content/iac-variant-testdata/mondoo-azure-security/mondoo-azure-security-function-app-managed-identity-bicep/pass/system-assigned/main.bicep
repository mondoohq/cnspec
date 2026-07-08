resource functionApp 'Microsoft.Web/sites@2023-12-01' = {
  name: 'func-prod-001'
  location: 'eastus'
  kind: 'functionapp'
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    serverFarmId: resourceId('Microsoft.Web/serverfarms', 'plan-prod-001')
    httpsOnly: true
  }
}
