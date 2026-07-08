resource diag 'Microsoft.Insights/diagnosticSettings@2021-05-01-preview' = {
  name: 'send-to-storage'
  properties: {
    storageAccountId: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-monitoring/providers/Microsoft.Storage/storageAccounts/auditlogsa'
    logs: [
      {
        category: 'Administrative'
        enabled: true
      }
    ]
  }
}
