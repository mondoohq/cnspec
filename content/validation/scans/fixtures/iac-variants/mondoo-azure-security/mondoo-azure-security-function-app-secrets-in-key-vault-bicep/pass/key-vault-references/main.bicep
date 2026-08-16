resource functionApp 'Microsoft.Web/sites@2023-01-01' = {
  name: 'event-processor'
  location: 'eastus'
  kind: 'functionapp'
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    siteConfig: {
      appSettings: [
        {
          name: 'FUNCTIONS_WORKER_RUNTIME'
          value: 'python'
        }
        {
          name: 'DB_CONNECTIONSTRING'
          value: '@Microsoft.KeyVault(SecretUri=https://example-kv.vault.azure.net/secrets/db-connection/)'
        }
        {
          name: 'PARTNER_APIKEY'
          value: '@Microsoft.KeyVault(SecretUri=https://example-kv.vault.azure.net/secrets/partner-api-key/)'
        }
      ]
    }
  }
}
