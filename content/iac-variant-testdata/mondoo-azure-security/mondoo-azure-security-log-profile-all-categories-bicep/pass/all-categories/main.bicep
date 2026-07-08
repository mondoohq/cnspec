resource logProfile 'Microsoft.Insights/logprofiles@2016-03-01' = {
  name: 'default'
  properties: {
    categories: [
      'Write'
      'Delete'
      'Action'
    ]
    locations: [
      'global'
      'eastus'
    ]
    retentionPolicy: {
      enabled: true
      days: 365
    }
  }
}
