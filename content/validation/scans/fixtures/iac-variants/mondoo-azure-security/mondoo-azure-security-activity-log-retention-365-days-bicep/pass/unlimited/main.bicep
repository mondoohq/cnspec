resource logProfile 'Microsoft.Insights/logprofiles@2016-03-01' = {
  name: 'default'
  properties: {
    storageAccountId: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-logs/providers/Microsoft.Storage/storageAccounts/contosologs'
    locations: [
      'eastus'
      'global'
    ]
    categories: [
      'Write'
      'Delete'
      'Action'
    ]
    retentionPolicy: {
      enabled: true
      days: 0
    }
  }
}
