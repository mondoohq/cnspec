resource redisFirewall 'Microsoft.Cache/redis/firewallRules@2023-08-01' = {
  name: 'app_tier'
  properties: {
    startIP: '10.0.1.0'
    endIP: '10.0.1.255'
  }
}
