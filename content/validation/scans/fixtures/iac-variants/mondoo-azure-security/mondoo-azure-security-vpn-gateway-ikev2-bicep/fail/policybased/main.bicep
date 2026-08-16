@description('Name of the VPN gateway')
param gatewayName string = 'vpn-gw-policybased'

@description('Deployment location')
param location string = resourceGroup().location

resource publicIp 'Microsoft.Network/publicIPAddresses@2023-09-01' = {
  name: '${gatewayName}-pip'
  location: location
  sku: {
    name: 'Basic'
  }
  properties: {
    publicIPAllocationMethod: 'Dynamic'
  }
}

// PolicyBased VPN only supports IKEv1, not IKEv2.
resource vnetGateway 'Microsoft.Network/virtualNetworkGateways@2023-09-01' = {
  name: gatewayName
  location: location
  properties: {
    gatewayType: 'Vpn'
    vpnType: 'PolicyBased'
    vpnGatewayGeneration: 'Generation1'
    sku: {
      name: 'Basic'
      tier: 'Basic'
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
