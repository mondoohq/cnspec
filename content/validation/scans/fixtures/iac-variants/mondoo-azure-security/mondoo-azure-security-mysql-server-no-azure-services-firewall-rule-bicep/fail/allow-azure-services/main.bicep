// 0.0.0.0 to 0.0.0.0 is the "Allow access to Azure services" rule. It admits
// every Azure tenant, not just this subscription.
resource firewallRule 'Microsoft.DBforMySQL/servers/firewallRules@2017-12-01' = {
  name: 'AllowAllWindowsAzureIps'
  properties: {
    startIpAddress: '0.0.0.0'
    endIpAddress: '0.0.0.0'
  }
}
