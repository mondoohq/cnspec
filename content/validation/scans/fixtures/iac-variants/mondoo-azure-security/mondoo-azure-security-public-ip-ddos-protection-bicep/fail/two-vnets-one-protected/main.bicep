resource ddosPlan 'Microsoft.Network/ddosProtectionPlans@2023-09-01' = {
  name: 'ddos-plan'
  location: 'eastus'
}

resource protectedVnet 'Microsoft.Network/virtualNetworks@2023-09-01' = {
  name: 'protected-vnet'
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

resource unprotectedVnet 'Microsoft.Network/virtualNetworks@2023-09-01' = {
  name: 'unprotected-vnet'
  location: 'eastus'
  properties: {
    addressSpace: {
      addressPrefixes: [
        '10.1.0.0/16'
      ]
    }
    subnets: [
      {
        name: 'workload'
        properties: {
          addressPrefix: '10.1.1.0/24'
        }
      }
    ]
  }
}

resource publicIp 'Microsoft.Network/publicIPAddresses@2023-09-01' = {
  name: 'pip-workload'
  location: 'eastus'
  sku: {
    name: 'Standard'
    tier: 'Regional'
  }
  properties: {
    publicIPAllocationMethod: 'Static'
    ddosSettings: {
      protectionMode: 'VirtualNetworkInherited'
    }
  }
}

resource nic 'Microsoft.Network/networkInterfaces@2023-09-01' = {
  name: 'nic-workload'
  location: 'eastus'
  properties: {
    ipConfigurations: [
      {
        name: 'primary'
        properties: {
          subnet: {
            id: '${unprotectedVnet.id}/subnets/workload'
          }
          privateIPAllocationMethod: 'Dynamic'
          publicIPAddress: {
            id: publicIp.id
          }
        }
      }
    ]
  }
}
