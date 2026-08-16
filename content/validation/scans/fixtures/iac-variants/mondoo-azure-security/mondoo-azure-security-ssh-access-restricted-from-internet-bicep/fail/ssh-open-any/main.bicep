resource nsg 'Microsoft.Network/networkSecurityGroups@2023-09-01' = {
  name: 'web-tier-nsg'
  location: 'eastus'
  properties: {
    securityRules: [
      {
        name: 'AllowSSHFromAnywhere'
        properties: {
          priority: 100
          direction: 'Inbound'
          access: 'Allow'
          protocol: 'Tcp'
          sourceAddressPrefix: '*'
          sourcePortRange: '*'
          destinationAddressPrefix: '*'
          destinationPortRange: '22'
        }
      }
    ]
  }
}
