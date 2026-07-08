resource containerGroup 'Microsoft.ContainerInstance/containerGroups@2023-05-01' = {
  name: 'contoso-jobs'
  location: 'eastus'
  properties: {
    osType: 'Linux'
    restartPolicy: 'OnFailure'
    containers: [
      {
        name: 'worker'
        properties: {
          image: 'nginx:latest'
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
