resource nsg 'Microsoft.Network/networkSecurityGroups@2023-09-01' = {
  name: 'mgmt-tier-nsg'
  location: 'eastus'
  properties: {
    securityRules: [
      {
        name: 'AllowManagementSSH'
        properties: {
          priority: 120
          direction: 'Inbound'
          access: 'Allow'
          protocol: '*'
          sourceAddressPrefix: 'Internet'
          sourcePortRange: '*'
          destinationAddressPrefix: '*'
          destinationPortRange: '22'
        }
      }
    ]
  }
}
