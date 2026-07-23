resource fwPolicy 'Microsoft.Network/firewallPolicies@2023-11-01' = {
  name: 'fwpol-prod-001'
  location: 'eastus'
  properties: {
    threatIntelMode: 'Deny'
  }
}

resource firewall 'Microsoft.Network/azureFirewalls@2023-11-01' = {
  name: 'afw-prod-001'
  location: 'eastus'
  properties: {
    sku: {
      name: 'AZFW_VNet'
      tier: 'Premium'
    }
    threatIntelMode: 'Deny'
    firewallPolicy: {
      id: fwPolicy.id
    }
  }
}
