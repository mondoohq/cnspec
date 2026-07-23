resource recoveryVault 'Microsoft.RecoveryServices/vaults@2023-06-01' = {
  name: 'rsv-secondary-westus'
  location: 'westus'
  sku: {
    name: 'RS0'
    tier: 'Standard'
  }
  properties: {
    publicNetworkAccess: 'Disabled'
  }
}
