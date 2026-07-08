resource mysqlServer 'Microsoft.DBforMySQL/servers@2017-12-01' = {
  name: 'contoso-mysql'
  location: 'eastus'
  sku: {
    name: 'GP_Gen5_2'
    tier: 'GeneralPurpose'
  }
  properties: {
    administratorLogin: 'dbadmin'
    version: '5.7'
  }
}

resource allowAzure 'Microsoft.DBforMySQL/servers/firewallRules@2017-12-01' = {
  parent: mysqlServer
  name: 'AllowAllAzureServices'
  properties: {
    startIpAddress: '0.0.0.0'
    endIpAddress: '0.0.0.0'
  }
}
