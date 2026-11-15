resource appGateway 'Microsoft.Network/applicationGateways@2023-09-01' = {
  name: 'example-appgw'
  location: 'eastus'
  properties: {
    sku: {
      name: 'WAF_Medium'
      tier: 'WAF'
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
  }
}
