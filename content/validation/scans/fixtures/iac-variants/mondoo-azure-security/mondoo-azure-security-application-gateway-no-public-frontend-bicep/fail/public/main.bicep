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
        name: 'publicFrontend'
        properties: {
          publicIPAddress: {
            id: resourceId('Microsoft.Network/publicIPAddresses', 'example-appgw-pip')
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
