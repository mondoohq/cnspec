resource nsg 'Microsoft.Network/networkSecurityGroups@2023-09-01' = {
  name: 'db-nsg'
  location: 'eastus'
  properties: {
    securityRules: [
      {
        name: 'AllowSqlFromAnywhere'
        properties: {
          priority: 120
          direction: 'Inbound'
          access: 'Allow'
          protocol: '*'
          sourceAddressPrefix: '*'
          sourcePortRange: '*'
          destinationAddressPrefix: '*'
          destinationPortRange: '1433'
        }
      }
    ]
  }
}
