resource app 'Microsoft.App/containerApps@2024-03-01' = {
  name: 'contoso-api'
  location: 'eastus'
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    managedEnvironmentId: resourceId('Microsoft.App/managedEnvironments', 'prod-env')
    configuration: {
      ingress: {
        external: true
        targetPort: 8080
        allowInsecure: false
      }
    }
    template: {
      containers: [
        {
          name: 'api'
          image: 'contosoregistry.azurecr.io/api@sha256:2c3a8f5b7d1e4a9c6b0f3e2d1a8c7b6e5d4f3a2b1c0d9e8f7a6b5c4d3e2f1a0b'
        }
      ]
    }
  }
}
