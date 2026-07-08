resource app 'Microsoft.App/containerApps@2024-03-01' = {
  name: 'contoso-api'
  location: 'eastus'
  properties: {
    managedEnvironmentId: resourceId('Microsoft.App/managedEnvironments', 'prod-env')
    configuration: {
      ingress: {
        external: true
        targetPort: 8080
        allowInsecure: false
      }
      secrets: [
        {
          name: 'registry-password'
          value: 'placeholder'
        }
      ]
      registries: [
        {
          server: 'contosoregistry.azurecr.io'
          username: 'contosoregistry'
          passwordSecretRef: 'registry-password'
        }
      ]
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
