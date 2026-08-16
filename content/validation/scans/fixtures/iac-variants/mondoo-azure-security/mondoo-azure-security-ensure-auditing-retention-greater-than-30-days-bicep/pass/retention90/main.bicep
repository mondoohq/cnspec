resource sqlAudit 'Microsoft.Sql/servers/auditingSettings@2023-05-01-preview' = {
  name: 'sql-prod/default'
  properties: {
    state: 'Enabled'
    isAzureMonitorTargetEnabled: true
    retentionDays: 90
  }
}
