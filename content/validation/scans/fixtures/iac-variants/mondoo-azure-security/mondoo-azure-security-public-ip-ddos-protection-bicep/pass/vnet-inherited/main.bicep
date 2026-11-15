resource ddosPlan 'Microsoft.Network/ddosProtectionPlans@2023-09-01' = {
  name: 'ddos-plan'
  location: 'eastus'
}

resource vnet 'Microsoft.Network/virtualNetworks@2023-09-01' = {
  name: 'vnet-protected'
  location: 'eastus'
  properties: {
    addressSpace: {
      addressPrefixes: [
        '10.0.0.0/16'
      ]
    }
    enableDdosProtection: true
    ddosProtectionPlan: {
      id: ddosPlan.id
    }
  }
}

resource publicIp 'Microsoft.Network/publicIPAddresses@2023-09-01' = {
  name: 'pip-gateway'
  location: 'eastus'
  sku: {
    name: 'Standard'
    tier: 'Regional'
  }
  properties: {
    publicIPAllocationMethod: 'Static'
    publicIPAddressVersion: 'IPv4'
    ddosSettings: {
      protectionMode: 'VirtualNetworkInherited'
    }
  }
}
