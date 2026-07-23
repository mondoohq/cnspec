@description('Name of the SQL server')
param sqlServerName string = 'contoso-sql-legacy'

resource sqlAdAdmin 'Microsoft.Sql/servers/administrators@2023-05-01-preview' = {
  name: '${sqlServerName}/ActiveDirectory'
  properties: {
    login: 'sqladmins@contoso.com'
    sid: '8bf9d6b0-3a1c-4f2e-9d7a-1c2b3d4e5f60'
    tenantId: '11111111-2222-3333-4444-555555555555'
  }
}
