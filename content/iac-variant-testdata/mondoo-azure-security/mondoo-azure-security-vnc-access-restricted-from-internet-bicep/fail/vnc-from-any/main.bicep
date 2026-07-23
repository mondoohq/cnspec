resource nsg 'Microsoft.Network/networkSecurityGroups@2023-09-01' = {
  name: 'workload-nsg'
  location: 'eastus'
  properties: {
    securityRules: [
      {
        name: 'AllowVncInbound'
        properties: {
          priority: 110
          direction: 'Inbound'
          access: 'Allow'
          protocol: 'Tcp'
          sourceAddressPrefix: '*'
          sourcePortRange: '*'
          destinationAddressPrefix: '*'
          destinationPortRange: '5900'
        }
      }
    ]
  }
}
