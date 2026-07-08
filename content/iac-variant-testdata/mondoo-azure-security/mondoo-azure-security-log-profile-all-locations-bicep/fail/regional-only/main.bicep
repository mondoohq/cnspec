resource storage 'Microsoft.Storage/storageAccounts@2023-01-01' = {
  name: 'activitylogsa'
  location: 'eastus'
  sku: {
    name: 'Standard_LRS'
  }
  kind: 'StorageV2'
}

resource logProfile 'Microsoft.Insights/logprofiles@2016-03-01' = {
  name: 'default'
  properties: {
    storageAccountId: storage.id
    locations: [
      'eastus'
      'westus'
    ]
    categories: [
      'Write'
      'Delete'
      'Action'
    ]
    retentionPolicy: {
      enabled: true
      days: 365
    }
  }
}
