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

resource allowAll 'Microsoft.DBforPostgreSQL/servers/firewallRules@2017-12-01' = {
  parent: postgresServer
  name: 'AllowAll'
  properties: {
    startIpAddress: '0.0.0.0'
    endIpAddress: '255.255.255.255'
  }
}
