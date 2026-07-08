resource vault 'Microsoft.RecoveryServices/vaults@2023-06-01' = {
  name: 'rsv-prod-eastus-001'
  location: 'eastus'
  sku: {
    name: 'RS0'
    tier: 'Standard'
  }
  properties: {
    publicNetworkAccess: 'Disabled'
  }
}
