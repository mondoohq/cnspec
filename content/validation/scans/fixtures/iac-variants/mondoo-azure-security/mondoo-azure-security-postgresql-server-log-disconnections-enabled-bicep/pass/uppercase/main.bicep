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
    sslEnforcement: 'Enabled'
  }
}

resource logDisconnections 'Microsoft.DBforPostgreSQL/servers/configurations@2017-12-01' = {
  parent: postgresServer
  name: 'log_disconnections'
  properties: {
    value: 'ON'
    source: 'user-override'
  }
}
