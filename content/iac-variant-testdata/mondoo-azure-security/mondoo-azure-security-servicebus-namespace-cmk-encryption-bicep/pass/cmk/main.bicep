resource sbns 'Microsoft.ServiceBus/namespaces@2022-10-01-preview' = {
  name: 'contoso-sb-prod'
  location: 'eastus'
  sku: {
    name: 'Premium'
    tier: 'Premium'
  }
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    minimumTlsVersion: '1.2'
    encryption: {
      keySource: 'Microsoft.KeyVault'
      keyVaultProperties: [
        {
          keyName: 'sbkey'
          keyVaultUri: 'https://contoso-kv.vault.azure.net'
        }
      ]
    }
  }
}
