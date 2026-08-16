resource logProfile 'Microsoft.Insights/logprofiles@2016-03-01' = {
  name: 'default'
  properties: {
    categories: [
      'Write'
    ]
    locations: [
      'global'
    ]
    retentionPolicy: {
      enabled: true
      days: 365
    }
  }
}
