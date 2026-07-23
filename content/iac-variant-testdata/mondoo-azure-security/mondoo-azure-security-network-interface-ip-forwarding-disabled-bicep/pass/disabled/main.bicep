resource nic 'Microsoft.Network/networkInterfaces@2023-09-01' = {
  name: 'nic-app-prod-001'
  location: 'eastus'
  properties: {
    enableIPForwarding: false
    ipConfigurations: [
      {
        name: 'ipconfig1'
        properties: {
          privateIPAllocationMethod: 'Dynamic'
          subnet: {
            id: resourceId('Microsoft.Network/virtualNetworks/subnets', 'vnet-prod', 'subnet-app')
          }
        }
      }
    ]
  }
}
