resource fwPolicy 'Microsoft.Network/firewallPolicies@2023-11-01' = {
  name: 'fwpol-prod-001'
  location: 'eastus'
  properties: {
    sku: {
      tier: 'Standard'
    }
    threatIntelMode: 'Deny'
  }
}
