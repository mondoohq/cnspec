resource vaultPrivateEndpoint 'Microsoft.Network/privateEndpoints@2023-05-01' = {
  name: 'vault-pe'
  location: 'eastus'
  properties: {
    subnet: {
      id: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-net/providers/Microsoft.Network/virtualNetworks/vnet/subnets/pe-subnet'
    }
    privateLinkServiceConnections: [
      {
        name: 'vault-connection'
        properties: {
          privateLinkServiceId: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-kv/providers/Microsoft.KeyVault/vaults/contosovault'
          groupIds: [
            'vault'
          ]
        }
      }
    ]
  }
}
