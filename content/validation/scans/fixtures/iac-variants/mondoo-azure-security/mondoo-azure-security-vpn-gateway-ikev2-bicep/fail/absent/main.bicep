@description('Name of the VPN gateway')
param gatewayName string = 'vpn-gw-notype'

@description('Deployment location')
param location string = resourceGroup().location

// vpnType is omitted; the RouteBased (IKEv2) requirement is not explicitly met.
resource vnetGateway 'Microsoft.Network/virtualNetworkGateways@2023-09-01' = {
  name: gatewayName
  location: location
  properties: {
    gatewayType: 'Vpn'
    vpnGatewayGeneration: 'Generation1'
    sku: {
      name: 'VpnGw1'
      tier: 'VpnGw1'
    }
    ipConfigurations: [
      {
        name: 'default'
        properties: {
          privateIPAllocationMethod: 'Dynamic'
        }
      }
    ]
  }
}
