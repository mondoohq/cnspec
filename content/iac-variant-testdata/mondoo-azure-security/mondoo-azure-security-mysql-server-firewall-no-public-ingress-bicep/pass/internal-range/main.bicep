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

resource allowInternal 'Microsoft.DBforMySQL/servers/firewallRules@2017-12-01' = {
  parent: mysqlServer
  name: 'AllowCorpNetwork'
  properties: {
    startIpAddress: '10.0.0.1'
    endIpAddress: '10.0.0.255'
  }
}
