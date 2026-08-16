resource diag 'Microsoft.Insights/diagnosticSettings@2021-05-01-preview' = {
  name: 'orphaned-diag'
  properties: {
    logs: [
      {
        category: 'Administrative'
        enabled: true
      }
    ]
  }
}
