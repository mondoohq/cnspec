// The policy exists and is attached but is switched off, so no rule in it is
// evaluated. This reads as protected in an inventory while filtering nothing.
resource wafPolicy 'Microsoft.Network/ApplicationGatewayWebApplicationFirewallPolicies@2023-09-01' = {
  name: 'public-waf'
  location: 'eastus'
  properties: {
    policySettings: {
      state: 'Disabled'
      mode: 'Detection'
    }
    managedRules: {
      managedRuleSets: [
        {
          ruleSetType: 'OWASP'
          ruleSetVersion: '3.2'
        }
      ]
    }
  }
}
