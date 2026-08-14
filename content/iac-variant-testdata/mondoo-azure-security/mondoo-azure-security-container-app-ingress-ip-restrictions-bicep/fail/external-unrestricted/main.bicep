// Externally reachable with no IP restrictions at all.
resource api 'Microsoft.App/containerApps@2024-03-01' = {
  name: 'api'
  location: 'eastus'
  properties: {
    configuration: {
      ingress: {
        external: true
        targetPort: 8080
      }
    }
  }
}
