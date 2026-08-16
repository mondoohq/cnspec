resource cosmos 'Microsoft.DocumentDB/databaseAccounts@2024-05-15' = {
  name: 'contoso-cosmos'
  location: 'eastus'
  kind: 'GlobalDocumentDB'
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    databaseAccountOfferType: 'Standard'
    keyVaultKeyUri: 'https://contoso-kv.vault.azure.net/keys/cosmos-cmk/9f8e7d6c5b4a3210'
    consistencyPolicy: {
      defaultConsistencyLevel: 'Session'
    }
    locations: [
      {
        locationName: 'eastus'
        failoverPriority: 0
        isZoneRedundant: false
      }
    ]
  }
}
