resource containerGroup 'Microsoft.ContainerInstance/containerGroups@2023-05-01' = {
  name: 'contoso-jobs'
  location: 'eastus'
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    osType: 'Linux'
    restartPolicy: 'OnFailure'
    imageRegistryCredentials: [
      {
        server: 'contosoregistry.azurecr.io'
        identity: 'system'
      }
    ]
    containers: [
      {
        name: 'worker'
        properties: {
          image: 'contosoregistry.azurecr.io/worker:3.2.1'
          resources: {
            requests: {
              cpu: 1
              memoryInGB: json('1.5')
            }
          }
        }
      }
    ]
  }
}
