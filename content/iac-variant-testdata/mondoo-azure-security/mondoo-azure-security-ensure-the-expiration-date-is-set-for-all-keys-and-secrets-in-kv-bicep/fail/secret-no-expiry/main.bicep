resource vault 'Microsoft.KeyVault/vaults@2023-07-01' existing = {
  name: 'contoso-kv'
}

resource dbSecret 'Microsoft.KeyVault/vaults/secrets@2023-07-01' = {
  parent: vault
  name: 'db-connection-string'
  properties: {
    contentType: 'text/plain'
    attributes: {
      enabled: true
    }
  }
}
