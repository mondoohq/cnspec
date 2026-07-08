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

resource logCheckpoints 'Microsoft.DBforPostgreSQL/servers/configurations@2017-12-01' = {
  parent: postgresServer
  name: 'log_checkpoints'
  properties: {
    value: 'off'
    source: 'user-override'
  }
}
