// 0.0.0.0 through 255.255.255.255 is the whole IPv4 space, which is the same
// as having no firewall at all.
resource redisFirewall 'Microsoft.Cache/redis/firewallRules@2023-08-01' = {
  name: 'prod-cache/allow_all'
  properties: {
    startIP: '0.0.0.0'
    endIP: '255.255.255.255'
  }
}
