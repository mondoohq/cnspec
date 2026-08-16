// TLS terminates at the gateway and the hop to the backend pool is plaintext.
resource appgw 'Microsoft.Network/applicationGateways@2023-09-01' = {
  name: 'public-appgw'
  location: 'eastus'
  properties: {
    sku: {
      name: 'WAF_v2'
      tier: 'WAF_v2'
      capacity: 2
    }
    backendHttpSettingsCollection: [
      {
        name: 'app-http'
        properties: {
          port: 80
          protocol: 'Http'
          cookieBasedAffinity: 'Disabled'
        }
      }
    ]
    httpListeners: [
      {
        name: 'https-listener'
        properties: {
          protocol: 'Https'
        }
      }
    ]
  }
}
