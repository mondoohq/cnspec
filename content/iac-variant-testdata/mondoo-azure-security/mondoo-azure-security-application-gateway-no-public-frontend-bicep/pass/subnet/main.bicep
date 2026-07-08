resource appGateway 'Microsoft.Network/applicationGateways@2023-09-01' = {
  name: 'example-appgw'
  location: 'eastus'
  properties: {
    sku: {
      name: 'Standard_v2'
      tier: 'Standard_v2'
      capacity: 2
    }
    gatewayIPConfigurations: [
      {
        name: 'appGatewayIpConfig'
        properties: {
          subnet: {
            id: resourceId('Microsoft.Network/virtualNetworks/subnets', 'example-vnet', 'appgw-subnet')
          }
        }
      }
    ]
    frontendIPConfigurations: [
      {
        name: 'privateFrontend'
        properties: {
          privateIPAllocationMethod: 'Static'
          privateIPAddress: '10.0.1.10'
          subnet: {
            id: resourceId('Microsoft.Network/virtualNetworks/subnets', 'example-vnet', 'appgw-subnet')
          }
        }
      }
    ]
    frontendPorts: [
      {
        name: 'httpsPort'
        properties: {
          port: 443
        }
      }
    ]
  }
}
