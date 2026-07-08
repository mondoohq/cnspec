resource firewall 'Microsoft.Network/azureFirewalls@2023-11-01' = {
  name: 'afw-prod-001'
  location: 'eastus'
  properties: {
    threatIntelMode: 'Deny'
  }
  sku: {
    name: 'AZFW_VNet'
    tier: 'Standard'
  }
}
