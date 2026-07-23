resource mysqlServer 'Microsoft.DBforMySQL/servers@2017-12-01' = {
  name: 'contoso-mysql-legacy'
  location: 'eastus'
  sku: {
    name: 'GP_Gen5_2'
    tier: 'GeneralPurpose'
  }
  properties: {
    version: '5.7'
    administratorLogin: 'mysqladmin'
    sslEnforcement: 'Enabled'
    minimalTlsVersion: 'TLS1_2'
  }
}
