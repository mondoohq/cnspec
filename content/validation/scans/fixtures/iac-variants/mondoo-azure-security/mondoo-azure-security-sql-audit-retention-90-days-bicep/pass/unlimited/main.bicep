resource sqlAudit 'Microsoft.Sql/servers/auditingSettings@2023-05-01-preview' = {
  name: 'contoso-sql-prod/default'
  properties: {
    state: 'Enabled'
    isAzureMonitorTargetEnabled: true
    storageEndpoint: 'https://contosoauditlogs.blob.core.windows.net/'
    retentionDays: 0
  }
}
