resource kusto 'Microsoft.Kusto/clusters@2023-08-15' = {
  name: 'analytics'
  location: 'eastus'
  sku: {
    name: 'Standard_D13_v2'
    tier: 'Standard'
    capacity: 2
  }
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    keyVaultProperties: {
      keyName: 'kusto-cmk'
      keyVaultUri: 'https://example-kv.vault.azure.net/'
      keyVersion: '0123456789abcdef0123456789abcdef'
    }
  }
}
