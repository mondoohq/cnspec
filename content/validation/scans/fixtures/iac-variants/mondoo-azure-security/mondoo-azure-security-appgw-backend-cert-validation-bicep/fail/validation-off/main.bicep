// HTTPS to the backend, but the certificate chain is not verified, so the
// encrypted hop can be man-in-the-middled inside the VNet.
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
        name: 'app-https'
        properties: {
          port: 443
          protocol: 'Https'
          cookieBasedAffinity: 'Disabled'
          validateCertChainAndExpiry: false
        }
      }
    ]
  }
}
