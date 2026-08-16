resource diag 'Microsoft.Insights/diagnosticSettings@2021-05-01-preview' = {
  name: 'send-to-law'
  properties: {
    workspaceId: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-monitoring/providers/Microsoft.OperationalInsights/workspaces/law-prod'
    logs: [
      {
        category: 'Administrative'
        enabled: true
      }
    ]
  }
}
