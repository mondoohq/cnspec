resource diskPrivateEndpoint 'Microsoft.Network/privateEndpoints@2023-05-01' = {
  name: 'disk-pe'
  location: 'eastus'
  properties: {
    subnet: {
      id: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-net/providers/Microsoft.Network/virtualNetworks/vnet/subnets/pe-subnet'
    }
    privateLinkServiceConnections: [
      {
        name: 'disk-connection'
        properties: {
          privateLinkServiceId: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-compute/providers/Microsoft.Compute/diskAccesses/disk-access-01'
          groupIds: [
            'disks'
          ]
        }
      }
    ]
  }
}
