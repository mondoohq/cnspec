resource api 'Microsoft.App/containerApps@2024-03-01' = {
  name: 'api'
  location: 'eastus'
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    configuration: {
      ingress: {
        external: false
        targetPort: 8080
      }
    }
  }
}
