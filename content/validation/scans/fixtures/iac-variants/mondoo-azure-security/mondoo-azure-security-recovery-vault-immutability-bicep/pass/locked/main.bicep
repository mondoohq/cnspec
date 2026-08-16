resource recoveryVault 'Microsoft.RecoveryServices/vaults@2023-06-01' = {
  name: 'rsv-compliance-eastus'
  location: 'eastus'
  sku: {
    name: 'RS0'
    tier: 'Standard'
  }
  properties: {
    securitySettings: {
      immutabilitySettings: {
        state: 'Locked'
      }
    }
  }
}
