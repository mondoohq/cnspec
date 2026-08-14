// A "*" origin lets any site on the internet make cross-origin calls to the
// API with the browser's ambient credentials.
resource api 'Microsoft.App/containerApps@2024-03-01' = {
  name: 'api'
  location: 'eastus'
  properties: {
    configuration: {
      ingress: {
        external: true
        targetPort: 8080
        corsPolicy: {
          allowedOrigins: [
            '*'
          ]
        }
      }
    }
  }
}
