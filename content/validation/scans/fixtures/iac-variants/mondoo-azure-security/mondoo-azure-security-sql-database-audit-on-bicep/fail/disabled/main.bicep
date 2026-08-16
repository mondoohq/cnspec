resource dbAudit 'Microsoft.Sql/servers/databases/auditingSettings@2023-05-01-preview' = {
  name: 'contoso-sql-prod/appdb/default'
  properties: {
    state: 'Disabled'
  }
}
