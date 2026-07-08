@description('Name of the Cosmos DB account')
param accountName string = 'contoso-cosmos-multiwrite'

resource cosmos 'Microsoft.DocumentDB/databaseAccounts@2024-05-15' = {
  name: accountName
  location: 'eastus'
  kind: 'GlobalDocumentDB'
  properties: {
    databaseAccountOfferType: 'Standard'
    enableMultipleWriteLocations: true
    consistencyPolicy: {
      defaultConsistencyLevel: 'Session'
    }
    locations: [
      {
        locationName: 'eastus'
        failoverPriority: 0
        isZoneRedundant: false
      }
      {
        locationName: 'westus'
        failoverPriority: 1
        isZoneRedundant: false
      }
    ]
  }
}
