resource batchAccount 'Microsoft.Batch/batchAccounts@2024-02-01' = {
  name: 'mybatchaccount'
  location: 'eastus'
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    encryption: {
      keySource: 'Microsoft.KeyVault'
      keyVaultProperties: {
        keyIdentifier: 'https://mykeyvault.vault.azure.net/keys/batchkey/abc123'
      }
    }
  }
}
