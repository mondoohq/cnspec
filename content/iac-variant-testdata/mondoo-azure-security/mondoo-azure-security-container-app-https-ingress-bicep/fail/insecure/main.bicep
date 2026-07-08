resource app 'Microsoft.App/containerApps@2024-03-01' = {
  name: 'legacy-api'
  location: 'eastus'
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    managedEnvironmentId: resourceId('Microsoft.App/managedEnvironments', 'prod-env')
    configuration: {
      ingress: {
        external: true
        targetPort: 80
        allowInsecure: true
        traffic: [
          {
            latestRevision: true
            weight: 100
          }
        ]
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
