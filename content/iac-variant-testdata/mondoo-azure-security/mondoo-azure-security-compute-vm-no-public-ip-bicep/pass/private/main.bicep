resource nic 'Microsoft.Network/networkInterfaces@2023-05-01' = {
  name: 'app-vm-01-nic'
  location: 'eastus'
  properties: {
    ipConfigurations: [
      {
        name: 'ipconfig1'
        properties: {
          privateIPAllocationMethod: 'Dynamic'
          subnet: {
            id: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-net/providers/Microsoft.Network/virtualNetworks/vnet/subnets/app-subnet'
          }
        }
      }
    ]
  }
}
