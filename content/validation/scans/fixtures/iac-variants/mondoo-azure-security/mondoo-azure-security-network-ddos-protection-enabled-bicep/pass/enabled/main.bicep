resource ddosPlan 'Microsoft.Network/ddosProtectionPlans@2023-09-01' = {
  name: 'ddos-prod-plan'
  location: 'eastus'
}

resource vnet 'Microsoft.Network/virtualNetworks@2023-09-01' = {
  name: 'vnet-prod-001'
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
    subnets: [
      {
        name: 'default'
        properties: {
          addressPrefix: '10.0.0.0/24'
        }
      }
    ]
  }
}
