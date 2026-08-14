// Internal-only ingress needs no IP allow list: it is not reachable from
// outside the container app environment.
resource worker 'Microsoft.App/containerApps@2024-03-01' = {
  name: 'worker'
  location: 'eastus'
  properties: {
    configuration: {
      ingress: {
        external: false
        targetPort: 8080
      }
    }
  }
}
