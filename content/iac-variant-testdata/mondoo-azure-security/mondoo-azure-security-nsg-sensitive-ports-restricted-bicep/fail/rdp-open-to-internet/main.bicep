resource nsg 'Microsoft.Network/networkSecurityGroups@2023-09-01' = {
  name: 'db-tier-nsg'
  location: 'eastus'
  properties: {
    securityRules: [
      {
        name: 'AllowMsSqlFromInternet'
        properties: {
          priority: 100
          direction: 'Inbound'
          access: 'Allow'
          protocol: 'Tcp'
          sourceAddressPrefix: '*'
          sourcePortRange: '*'
          destinationAddressPrefix: '*'
          destinationPortRange: '1433'
        }
      }
    ]
  }
}
