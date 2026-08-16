resource firewall 'Microsoft.Network/azureFirewalls@2023-11-01' = {
  name: 'afw-prod-001'
  location: 'eastus'
  properties: {
    firewallPolicy: {
      id: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-prod/providers/Microsoft.Network/firewallPolicies/fwpol-prod-001'
    }
  }
  sku: {
    name: 'AZFW_VNet'
    tier: 'Premium'
  }
}
