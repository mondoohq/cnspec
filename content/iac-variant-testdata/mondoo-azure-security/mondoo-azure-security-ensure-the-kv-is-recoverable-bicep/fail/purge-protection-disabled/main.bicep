resource vault 'Microsoft.KeyVault/vaults@2023-07-01' = {
  name: 'contoso-kv'
  location: 'eastus'
  properties: {
    tenantId: '00000000-0000-0000-0000-000000000000'
    sku: {
      family: 'A'
      name: 'standard'
    }
    enableSoftDelete: true
    softDeleteRetentionInDays: 90
    enablePurgeProtection: false
  }
}
