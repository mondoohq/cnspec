resource functionApp 'Microsoft.Web/sites@2023-12-01' = {
  name: 'func-prod-001'
  location: 'eastus'
  kind: 'functionapp'
  properties: {
    serverFarmId: resourceId('Microsoft.Web/serverfarms', 'plan-prod-001')
    httpsOnly: true
    publicNetworkAccess: 'Disabled'
  }
}

resource functionPrivateEndpoint 'Microsoft.Network/privateEndpoints@2023-09-01' = {
  name: 'func-pe'
  location: 'eastus'
  properties: {
    subnet: {
      id: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/vnet/subnets/pe-subnet'
    }
    privateLinkServiceConnections: [
      {
        name: 'func-connection'
        properties: {
          privateLinkServiceId: functionApp.id
          groupIds: [
            'sites'
          ]
        }
      }
    ]
  }
}
