resource acrPrivateEndpoint 'Microsoft.Network/privateEndpoints@2023-05-01' = {
  name: 'acr-pe'
  location: 'eastus'
  properties: {
    subnet: {
      id: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-net/providers/Microsoft.Network/virtualNetworks/vnet/subnets/pe-subnet'
    }
    privateLinkServiceConnections: [
      {
        name: 'acr-connection'
        properties: {
          privateLinkServiceId: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-acr/providers/Microsoft.ContainerRegistry/registries/contosoregistry'
          groupIds: [
            'registry'
          ]
        }
      }
    ]
  }
}
