resource nsg 'Microsoft.Network/networkSecurityGroups@2023-09-01' = {
  name: 'app-tier-nsg'
  location: 'eastus'
  properties: {
    securityRules: [
      {
        name: 'AllowAllOutbound'
        properties: {
          priority: 200
          direction: 'Outbound'
          access: 'Allow'
          protocol: '*'
          sourceAddressPrefix: '*'
          sourcePortRange: '*'
          destinationAddressPrefix: '*'
          destinationPortRange: '*'
        }
      }
    ]
  }
}
