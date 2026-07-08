resource privateEndpoint 'Microsoft.Network/privateEndpoints@2023-09-01' = {
  name: 'pe-storage-prod-001'
  location: 'eastus'
  properties: {
    subnet: {
      id: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-prod/providers/Microsoft.Network/virtualNetworks/vnet-prod/subnets/snet-endpoints'
    }
    privateLinkServiceConnections: [
      {
        name: 'plsc-storage-blob'
        properties: {
          privateLinkServiceId: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-prod/providers/Microsoft.Storage/storageAccounts/stprod001'
          groupIds: [
            'blob'
          ]
        }
      }
    ]
  }
}
