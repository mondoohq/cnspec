resource nsg 'Microsoft.Network/networkSecurityGroups@2023-09-01' = {
  name: 'workload-nsg'
  location: 'eastus'
  properties: {
    securityRules: [
      {
        name: 'AllowVncFromManagementSubnet'
        properties: {
          priority: 120
          direction: 'Inbound'
          access: 'Allow'
          protocol: 'Tcp'
          sourceAddressPrefix: '10.0.1.0/24'
          sourcePortRange: '*'
          destinationAddressPrefix: '*'
          destinationPortRange: '5900'
        }
      }
    ]
  }
}
