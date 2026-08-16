resource sqlFirewall 'Microsoft.Sql/servers/firewallRules@2023-05-01-preview' = {
  name: 'sql-prod-eastus/AllowCorpNetwork'
  properties: {
    startIpAddress: '203.0.113.10'
    endIpAddress: '203.0.113.20'
  }
}
