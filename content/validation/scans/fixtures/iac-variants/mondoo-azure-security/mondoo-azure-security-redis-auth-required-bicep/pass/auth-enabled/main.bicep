resource redis 'Microsoft.Cache/redis@2023-08-01' = {
  name: 'redis-prod-eastus-001'
  location: 'eastus'
  properties: {
    sku: {
      name: 'Standard'
      family: 'C'
      capacity: 1
    }
    enableNonSslPort: false
    minimumTlsVersion: '1.2'
    redisConfiguration: {
      'maxmemory-policy': 'allkeys-lru'
    }
  }
}
