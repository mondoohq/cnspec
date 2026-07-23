resource functionApp 'Microsoft.Web/sites@2022-09-01' = {
  name: 'func-prod-001'
  location: 'eastus'
  kind: 'functionapp'
  properties: {
    serverFarmId: resourceId('Microsoft.Web/serverfarms', 'plan-prod-001')
    httpsOnly: true
    clientCertEnabled: true
    clientCertMode: 'Optional'
  }
}
