// Identity type None means the app authenticates to Azure services with
// long-lived secrets carried in configuration instead.
resource api 'Microsoft.App/containerApps@2024-03-01' = {
  name: 'api'
  location: 'eastus'
  identity: {
    type: 'None'
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
