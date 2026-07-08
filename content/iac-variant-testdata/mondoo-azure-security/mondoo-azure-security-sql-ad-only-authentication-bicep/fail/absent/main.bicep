@description('Name of the SQL server')
param sqlServerName string = 'contoso-sql-sqlauth'

@description('Deployment location')
param location string = resourceGroup().location

resource sqlServer 'Microsoft.Sql/servers@2023-05-01-preview' = {
  name: sqlServerName
  location: location
  properties: {
    version: '12.0'
    minimalTlsVersion: '1.2'
    administratorLogin: 'sqladmin'
    administratorLoginPassword: 'P@ssw0rd-ChangeMe!'
  }
}
