resource appgw 'Microsoft.Network/applicationGateways@2024-10-01' = {
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
        name: 'app-https'
        properties: {
          port: 443
          protocol: 'Https'
          cookieBasedAffinity: 'Disabled'
          validateCertChainAndExpiry: true
        }
      }
    ]
  }
}
