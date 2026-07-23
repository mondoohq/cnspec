@description('Name of the SQL server')
param sqlServerName string = 'contoso-sql-prod'

@description('Deployment location')
param location string = resourceGroup().location

resource sqlServer 'Microsoft.Sql/servers@2023-05-01-preview' = {
  name: sqlServerName
  location: location
  properties: {
    version: '12.0'
    minimalTlsVersion: '1.2'
    publicNetworkAccess: 'Disabled'
    administrators: {
      administratorType: 'ActiveDirectory'
      principalType: 'Group'
      login: 'sqladmins@contoso.com'
      sid: '8bf9d6b0-3a1c-4f2e-9d7a-1c2b3d4e5f60'
      tenantId: '11111111-2222-3333-4444-555555555555'
      azureADOnlyAuthentication: true
    }
  }
}
