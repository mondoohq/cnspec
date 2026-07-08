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
    network: {
      publicNetworkAccess: 'Enabled'
    }
    storage: {
      storageSizeGB: 128
    }
  }
}
