resource vault 'Microsoft.KeyVault/vaults@2023-07-01' existing = {
  name: 'contoso-kv'
}

resource encryptionKey 'Microsoft.KeyVault/vaults/keys@2023-07-01' = {
  parent: vault
  name: 'app-encryption-key'
  properties: {
    kty: 'RSA'
    keySize: 2048
    attributes: {
      enabled: true
    }
  }
}
