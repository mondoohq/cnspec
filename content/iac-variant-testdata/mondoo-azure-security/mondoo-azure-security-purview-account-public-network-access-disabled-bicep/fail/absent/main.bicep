resource purview 'Microsoft.Purview/accounts@2021-12-01' = {
  name: 'purviewaccount'
  location: 'eastus'
  identity: {
    type: 'SystemAssigned'
  }
  sku: {
    name: 'Standard'
    capacity: 1
  }
  properties: {
    managedResourceGroupName: 'managed-rg-purview'
  }
}
