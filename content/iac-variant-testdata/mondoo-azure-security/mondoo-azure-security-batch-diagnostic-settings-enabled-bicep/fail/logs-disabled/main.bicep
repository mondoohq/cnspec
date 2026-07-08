resource batchAccount 'Microsoft.Batch/batchAccounts@2024-02-01' = {
  name: 'mybatchaccount'
  location: 'eastus'
  properties: {}
}

resource logAnalytics 'Microsoft.OperationalInsights/workspaces@2023-09-01' = {
  name: 'myworkspace'
  location: 'eastus'
}

resource batchDiagnostics 'Microsoft.Insights/diagnosticSettings@2021-05-01-preview' = {
  name: 'batchDiagnostics'
  scope: batchAccount
  properties: {
    workspaceId: logAnalytics.id
    logs: [
      {
        category: 'ServiceLog'
        enabled: false
        retentionPolicy: {
          enabled: false
          days: 0
        }
      }
    ]
    metrics: [
      {
        category: 'AllMetrics'
        enabled: true
      }
    ]
  }
}
