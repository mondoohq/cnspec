resource nsg 'Microsoft.Network/networkSecurityGroups@2023-09-01' = {
  name: 'app-tier-nsg'
  location: 'eastus'
  properties: {
    securityRules: [
      {
        name: 'DenySSHFromInternet'
        properties: {
          priority: 100
          direction: 'Inbound'
          access: 'Deny'
          protocol: 'Tcp'
          sourceAddressPrefix: 'Internet'
          sourcePortRange: '*'
          destinationAddressPrefix: '*'
          destinationPortRange: '22'
        }
      }
      {
        name: 'AllowHTTPSInbound'
        properties: {
          priority: 200
          direction: 'Inbound'
          access: 'Allow'
          protocol: 'Tcp'
          sourceAddressPrefix: 'Internet'
          sourcePortRange: '*'
          destinationAddressPrefix: '*'
          destinationPortRange: '443'
        }
      }
    ]
  }
}
