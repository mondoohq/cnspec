resource sqlFirewall 'Microsoft.Sql/servers/firewallRules@2023-05-01-preview' = {
  name: 'sql-prod-eastus/AllowAllWindowsAzureIps'
  properties: {
    startIpAddress: '0.0.0.0'
    endIpAddress: '0.0.0.0'
  }
}
