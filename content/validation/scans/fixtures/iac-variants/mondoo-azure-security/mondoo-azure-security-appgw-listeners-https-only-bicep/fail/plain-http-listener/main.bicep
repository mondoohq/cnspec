// A plain HTTP listener accepts unencrypted client traffic, including any
// credentials sent before a redirect could take effect.
resource appgw 'Microsoft.Network/applicationGateways@2023-09-01' = {
  name: 'public-appgw'
  location: 'eastus'
  properties: {
    sku: {
      name: 'WAF_v2'
      tier: 'WAF_v2'
      capacity: 2
    }
    httpListeners: [
      {
        name: 'https-listener'
        properties: {
          protocol: 'Https'
        }
      }
      {
        name: 'http-listener'
        properties: {
          protocol: 'Http'
        }
      }
    ]
  }
}
