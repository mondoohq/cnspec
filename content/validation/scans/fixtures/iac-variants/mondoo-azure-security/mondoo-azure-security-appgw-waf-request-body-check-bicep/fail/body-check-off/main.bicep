// With body inspection off the WAF only sees headers and the URL, so injection
// payloads delivered in a POST body pass straight through.
resource wafPolicy 'Microsoft.Network/ApplicationGatewayWebApplicationFirewallPolicies@2023-09-01' = {
  name: 'public-waf'
  location: 'eastus'
  properties: {
    policySettings: {
      state: 'Enabled'
      mode: 'Prevention'
      requestBodyCheck: false
    }
  }
}
