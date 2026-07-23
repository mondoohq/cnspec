resource hsmPrivateEndpoint 'Microsoft.Network/privateEndpoints@2023-05-01' = {
  name: 'hsm-pe'
  location: 'eastus'
  properties: {
    subnet: {
      id: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-net/providers/Microsoft.Network/virtualNetworks/vnet/subnets/pe-subnet'
    }
    privateLinkServiceConnections: [
      {
        name: 'hsm-connection'
        properties: {
          privateLinkServiceId: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-hsm/providers/Microsoft.KeyVault/managedHSMs/contosohsm'
          groupIds: [
            'managedhsm'
          ]
        }
      }
    ]
  }
}
