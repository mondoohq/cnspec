resource searchService 'Microsoft.Search/searchServices@2024-06-01-preview' = {
  name: 'example-search'
  location: 'eastus'
  sku: {
    name: 'standard'
  }
  properties: {
    replicaCount: 1
  }
}
