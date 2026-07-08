resource firewall 'Microsoft.Network/azureFirewalls@2023-11-01' = {
  name: 'afw-prod-001'
  location: 'eastus'
  properties: {
    sku: {
      name: 'AZFW_VNet'
      tier: 'Standard'
    }
    threatIntelMode: 'Alert'
    ipConfigurations: [
      {
        name: 'ipconfig1'
        properties: {
          publicIPAddress: {
            id: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-prod/providers/Microsoft.Network/publicIPAddresses/pip-afw'
          }
          subnet: {
            id: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-prod/providers/Microsoft.Network/virtualNetworks/vnet-prod/subnets/AzureFirewallSubnet'
          }
        }
      }
    ]
  }
}
