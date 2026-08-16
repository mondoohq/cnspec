resource mysql 'Microsoft.DBforMySQL/servers@2017-12-01' = {
  name: 'contoso-mysql-prod'
  location: 'eastus'
  sku: {
    name: 'GP_Gen5_2'
    tier: 'GeneralPurpose'
    family: 'Gen5'
    capacity: 2
  }
  properties: {
    administratorLogin: 'mysqladmin'
    administratorLoginPassword: 'REPLACE_WITH_KEYVAULT_REF'
    version: '5.7'
    sslEnforcement: 'Enabled'
    minimalTlsVersion: 'TLS1_2'
    publicNetworkAccess: 'Enabled'
  }
}
