resource nsg 'Microsoft.Network/networkSecurityGroups@2023-09-01' = {
  name: 'data-tier-nsg'
  location: 'eastus'
  properties: {
    securityRules: [
      {
        name: 'AllowMySqlFromInternet'
        properties: {
          priority: 110
          direction: 'Inbound'
          access: 'Allow'
          protocol: 'Tcp'
          sourceAddressPrefix: 'Internet'
          sourcePortRange: '*'
          destinationAddressPrefix: '*'
          destinationPortRange: '3306'
        }
      }
    ]
  }
}
