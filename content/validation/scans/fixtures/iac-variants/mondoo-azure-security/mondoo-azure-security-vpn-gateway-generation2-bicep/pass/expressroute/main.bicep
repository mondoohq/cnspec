@description('Name of the ExpressRoute gateway')
param gatewayName string = 'er-gw-prod'

@description('Deployment location')
param location string = resourceGroup().location

// ExpressRoute gateways are not VPN gateways, so the Generation2 requirement
// does not apply to them.
resource vnetGateway 'Microsoft.Network/virtualNetworkGateways@2023-09-01' = {
  name: gatewayName
  location: location
  properties: {
    gatewayType: 'ExpressRoute'
    sku: {
      name: 'ErGw1AZ'
      tier: 'ErGw1AZ'
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
