resource storageAccount 'Microsoft.Storage/storageAccounts@2023-01-01' = {
  name: 'stprodencrypt002'
  location: 'eastus'
  sku: {
    name: 'Standard_LRS'
  }
  kind: 'StorageV2'
  properties: {
    minimumTlsVersion: 'TLS1_2'
  }
}

resource encryptionScope 'Microsoft.Storage/storageAccounts/encryptionScopes@2023-01-01' = {
  parent: storageAccount
  name: 'securescope'
  properties: {
    source: 'Microsoft.KeyVault'
    keyVaultProperties: {
      keyUri: 'https://kv-prod-001.vault.azure.net/keys/storage-cmk/abcdef1234567890'
    }
    requireInfrastructureEncryption: true
  }
}
