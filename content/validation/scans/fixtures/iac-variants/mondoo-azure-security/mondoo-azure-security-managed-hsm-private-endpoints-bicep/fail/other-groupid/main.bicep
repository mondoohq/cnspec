resource hsm 'Microsoft.KeyVault/managedHSMs@2023-07-01' = {
  name: 'contosohsm'
  location: 'eastus'
  sku: {
    name: 'Standard_B1'
    family: 'B'
  }
  properties: {
    tenantId: '00000000-0000-0000-0000-000000000000'
    initialAdminObjectIds: [
      '11111111-1111-1111-1111-111111111111'
    ]
  }
}

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
