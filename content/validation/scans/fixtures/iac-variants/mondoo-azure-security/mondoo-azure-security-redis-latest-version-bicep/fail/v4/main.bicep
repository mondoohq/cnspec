@description('Name of the Redis cache')
param redisName string = 'contoso-redis-legacy'

@description('Deployment location')
param location string = resourceGroup().location

resource redis 'Microsoft.Cache/redis@2023-08-01' = {
  name: redisName
  location: location
  properties: {
    sku: {
      name: 'Standard'
      family: 'C'
      capacity: 1
    }
    redisVersion: '4'
    minimumTlsVersion: '1.2'
    enableNonSslPort: false
    publicNetworkAccess: 'Disabled'
  }
}
