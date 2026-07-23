resource recoveryVault 'Microsoft.RecoveryServices/vaults@2023-06-01' = {
  name: 'rsv-prod-eastus'
  location: 'eastus'
  sku: {
    name: 'RS0'
    tier: 'Standard'
  }
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    encryption: {
      keyVaultProperties: {
        keyUri: 'https://kv-prod.vault.azure.net/keys/rsv-cmk'
      }
      kekIdentity: {
        useSystemAssignedIdentity: true
      }
      infrastructureEncryption: 'Enabled'
    }
  }
}
