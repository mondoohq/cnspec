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
          image: 'contosoregistry.azurecr.io/api:1.4.0'
        }
      ]
    }
  }
}
