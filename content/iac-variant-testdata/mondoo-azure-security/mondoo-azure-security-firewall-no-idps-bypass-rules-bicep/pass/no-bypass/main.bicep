resource firewallPolicy 'Microsoft.Network/firewallPolicies@2023-09-01' = {
  name: 'prod-policy'
  location: 'eastus'
  properties: {
    sku: {
      tier: 'Premium'
    }
    intrusionDetection: {
      mode: 'Deny'
      configuration: {
        signatureOverrides: []
      }
    }
  }
}
