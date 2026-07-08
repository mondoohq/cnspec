resource vault 'Microsoft.KeyVault/vaults@2023-07-01' = {
  name: 'kv-prod-001'
  location: 'eastus'
  properties: {
    sku: {
      family: 'A'
      name: 'standard'
    }
    tenantId: subscription().tenantId
    enableRbacAuthorization: true
  }
}

resource encryptionKey 'Microsoft.KeyVault/vaults/keys@2023-07-01' = {
  parent: vault
  name: 'data-encryption-key'
  properties: {
    kty: 'RSA'
    keySize: 2048
  }
}
