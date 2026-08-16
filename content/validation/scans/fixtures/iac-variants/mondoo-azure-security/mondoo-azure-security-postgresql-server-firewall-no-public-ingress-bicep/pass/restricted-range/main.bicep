resource postgresServer 'Microsoft.DBforPostgreSQL/servers@2017-12-01' = {
  name: 'contoso-postgres'
  location: 'eastus'
  sku: {
    name: 'GP_Gen5_2'
    tier: 'GeneralPurpose'
    capacity: 2
    family: 'Gen5'
  }
  properties: {
    version: '11'
    administratorLogin: 'pgadmin'
    sslEnforcement: 'Enabled'
  }
}

resource officeAccess 'Microsoft.DBforPostgreSQL/servers/firewallRules@2017-12-01' = {
  parent: postgresServer
  name: 'AllowOfficeNetwork'
  properties: {
    startIpAddress: '203.0.113.0'
    endIpAddress: '203.0.113.255'
  }
}
