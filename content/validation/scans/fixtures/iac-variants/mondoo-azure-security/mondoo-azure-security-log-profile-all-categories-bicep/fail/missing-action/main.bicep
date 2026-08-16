resource logProfile 'Microsoft.Insights/logprofiles@2016-03-01' = {
  name: 'default'
  properties: {
    categories: [
      'Write'
      'Delete'
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
