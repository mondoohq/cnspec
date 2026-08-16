resource nic 'Microsoft.Network/networkInterfaces@2023-09-01' = {
  name: 'nic-nva-prod-001'
  location: 'eastus'
  properties: {
    enableIPForwarding: true
    ipConfigurations: [
      {
        name: 'ipconfig1'
        properties: {
          privateIPAllocationMethod: 'Dynamic'
          subnet: {
            id: resourceId('Microsoft.Network/virtualNetworks/subnets', 'vnet-prod', 'subnet-nva')
          }
        }
      }
    ]
  }
}
