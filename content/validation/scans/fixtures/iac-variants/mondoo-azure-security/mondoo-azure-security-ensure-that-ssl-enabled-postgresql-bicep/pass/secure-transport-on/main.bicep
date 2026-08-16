resource postgresServer 'Microsoft.DBforPostgreSQL/flexibleServers@2023-06-01-preview' = {
  name: 'contoso-postgres'
  location: 'eastus'
  sku: {
    name: 'Standard_D2ds_v4'
    tier: 'GeneralPurpose'
  }
  properties: {
    version: '15'
    administratorLogin: 'pgadmin'
    storage: {
      storageSizeGB: 128
    }
  }

  resource secureTransport 'configurations@2023-06-01-preview' = {
    name: 'require_secure_transport'
    properties: {
      value: 'on'
      source: 'user-override'
    }
  }
}
