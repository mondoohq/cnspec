resource firewallRule 'Microsoft.DBforMySQL/servers/firewallRules@2017-12-01' = {
  name: 'corporate-egress'
  properties: {
    startIpAddress: '203.0.113.0'
    endIpAddress: '203.0.113.255'
  }
}
