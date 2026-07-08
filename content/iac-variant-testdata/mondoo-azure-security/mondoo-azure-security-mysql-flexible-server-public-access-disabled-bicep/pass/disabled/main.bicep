resource mysql 'Microsoft.DBforMySQL/flexibleServers@2023-12-30' = {
  name: 'contoso-mysql-flex'
  location: 'eastus'
  sku: {
    name: 'Standard_D2ds_v4'
    tier: 'GeneralPurpose'
  }
  properties: {
    administratorLogin: 'dbadmin'
    version: '8.0.21'
    network: {
      publicNetworkAccess: 'Disabled'
    }
  }
}
