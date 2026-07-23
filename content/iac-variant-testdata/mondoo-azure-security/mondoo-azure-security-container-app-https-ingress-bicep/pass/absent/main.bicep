resource app 'Microsoft.App/containerApps@2024-03-01' = {
  name: 'contoso-web'
  location: 'eastus'
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    managedEnvironmentId: resourceId('Microsoft.App/managedEnvironments', 'prod-env')
    configuration: {
      ingress: {
        external: true
        targetPort: 443
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
          name: 'web'
          image: 'contosoregistry.azurecr.io/web:2.0.1'
        }
      ]
    }
  }
}
