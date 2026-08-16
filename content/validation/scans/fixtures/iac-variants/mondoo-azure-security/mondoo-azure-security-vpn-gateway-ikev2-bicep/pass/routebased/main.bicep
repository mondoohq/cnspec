@description('Name of the VPN gateway')
param gatewayName string = 'vpn-gw-routebased'

@description('Deployment location')
param location string = resourceGroup().location

resource publicIp 'Microsoft.Network/publicIPAddresses@2023-09-01' = {
  name: '${gatewayName}-pip'
  location: location
  sku: {
    name: 'Standard'
  }
  properties: {
    publicIPAllocationMethod: 'Static'
  }
}

resource vnetGateway 'Microsoft.Network/virtualNetworkGateways@2023-09-01' = {
  name: gatewayName
  location: location
  properties: {
    gatewayType: 'Vpn'
    vpnType: 'RouteBased'
    vpnGatewayGeneration: 'Generation2'
    sku: {
      name: 'VpnGw2'
      tier: 'VpnGw2'
    }
    ipConfigurations: [
      {
        name: 'default'
        properties: {
          privateIPAllocationMethod: 'Dynamic'
          publicIPAddress: {
            id: publicIp.id
          }
        }
      }
    ]
  }
}
