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

resource connThrottle 'Microsoft.DBforPostgreSQL/servers/configurations@2017-12-01' = {
  parent: postgresServer
  name: 'connection_throttling'
  properties: {
    value: 'on'
    source: 'user-override'
  }
}
