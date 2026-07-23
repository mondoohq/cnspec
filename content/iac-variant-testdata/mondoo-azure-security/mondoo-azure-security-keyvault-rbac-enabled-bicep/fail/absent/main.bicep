resource keyVault 'Microsoft.KeyVault/vaults@2023-07-01' = {
  name: 'kv-payments-old'
  location: 'eastus'
  properties: {
    sku: {
      family: 'A'
      name: 'standard'
    }
    tenantId: subscription().tenantId
    enableSoftDelete: true
    softDeleteRetentionInDays: 90
    accessPolicies: []
  }
}
