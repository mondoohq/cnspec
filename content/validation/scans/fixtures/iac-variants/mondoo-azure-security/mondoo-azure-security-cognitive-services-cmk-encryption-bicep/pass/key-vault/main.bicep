resource vision 'Microsoft.CognitiveServices/accounts@2023-05-01' = {
  name: 'vision'
  location: 'eastus'
  kind: 'ComputerVision'
  sku: {
    name: 'S1'
  }
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    encryption: {
      keySource: 'Microsoft.KeyVault'
      keyVaultProperties: {
        keyName: 'cognitive-cmk'
        keyVaultUri: 'https://example-kv.vault.azure.net/'
      }
    }
  }
}
