resource mysqlFirewall 'Microsoft.DBforMySQL/flexibleServers/firewallRules@2023-06-30' = {
  name: 'mysql-prod-eastus/AllowCorpNetwork'
  properties: {
    startIpAddress: '192.0.2.100'
    endIpAddress: '192.0.2.110'
  }
}
