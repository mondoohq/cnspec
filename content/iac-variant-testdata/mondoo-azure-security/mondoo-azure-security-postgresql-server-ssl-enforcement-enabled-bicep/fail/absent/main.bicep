resource postgresServer 'Microsoft.DBforPostgreSQL/servers@2017-12-01' = {
  name: 'contoso-pg'
  location: 'eastus'
  sku: {
    name: 'GP_Gen5_2'
    tier: 'GeneralPurpose'
  }
  properties: {
    version: '11'
    administratorLogin: 'pgadmin'
    minimalTlsVersion: 'TLS1_2'
    publicNetworkAccess: 'Disabled'
  }
}
