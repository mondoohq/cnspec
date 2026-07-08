resource sqlServer 'Microsoft.Sql/servers@2023-05-01-preview' = {
  name: 'sql-prod-eastus'
  location: 'eastus'
  properties: {
    administratorLogin: 'sqladmin'
    version: '12.0'
    publicNetworkAccess: 'Disabled'
  }
}
