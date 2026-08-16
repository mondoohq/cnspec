resource pgFirewall 'Microsoft.DBforPostgreSQL/flexibleServers/firewallRules@2023-06-01-preview' = {
  name: 'pg-prod-eastus/AllowCorpNetwork'
  properties: {
    startIpAddress: '198.51.100.5'
    endIpAddress: '198.51.100.15'
  }
}
