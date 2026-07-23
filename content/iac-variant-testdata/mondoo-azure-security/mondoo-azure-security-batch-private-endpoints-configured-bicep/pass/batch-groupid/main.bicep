resource batchAccount 'Microsoft.Batch/batchAccounts@2024-02-01' = {
  name: 'mybatchaccount'
  location: 'eastus'
  properties: {
    publicNetworkAccess: 'Disabled'
  }
}

resource batchPrivateEndpoint 'Microsoft.Network/privateEndpoints@2023-09-01' = {
  name: 'batch-pe'
  location: 'eastus'
  properties: {
    subnet: {
      id: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/vnet/subnets/pe-subnet'
    }
    privateLinkServiceConnections: [
      {
        name: 'batch-connection'
        properties: {
          privateLinkServiceId: batchAccount.id
          groupIds: [
            'batchAccount'
          ]
        }
      }
    ]
  }
}
