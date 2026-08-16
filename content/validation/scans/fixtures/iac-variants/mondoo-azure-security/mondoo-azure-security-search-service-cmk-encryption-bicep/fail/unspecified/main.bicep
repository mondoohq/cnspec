@description('Name of the Azure Cognitive Search service')
param searchServiceName string = 'contoso-search-dev'

@description('Deployment location')
param location string = resourceGroup().location

resource search 'Microsoft.Search/searchServices@2023-11-01' = {
  name: searchServiceName
  location: location
  sku: {
    name: 'standard'
  }
  properties: {
    replicaCount: 1
    partitionCount: 1
    hostingMode: 'default'
    encryptionWithCmk: {
      enforcement: 'Unspecified'
    }
  }
}
