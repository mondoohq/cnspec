resource cosmos 'Microsoft.DocumentDB/databaseAccounts@2024-05-15' = {
  name: 'contoso-cosmos'
  location: 'eastus'
  kind: 'GlobalDocumentDB'
  properties: {
    databaseAccountOfferType: 'Standard'
    consistencyPolicy: {
      defaultConsistencyLevel: 'Session'
    }
    cors: [
      {
        allowedOrigins: 'https://app.contoso.com'
        allowedMethods: 'GET,POST'
      }
    ]
    locations: [
      {
        locationName: 'eastus'
        failoverPriority: 0
        isZoneRedundant: false
      }
    ]
  }
}
