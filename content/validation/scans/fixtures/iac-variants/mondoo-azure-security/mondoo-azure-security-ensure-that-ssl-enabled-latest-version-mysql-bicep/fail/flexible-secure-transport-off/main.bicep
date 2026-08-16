resource mysqlServer 'Microsoft.DBforMySQL/flexibleServers@2023-06-30' = {
  name: 'contoso-mysql'
  location: 'eastus'
  sku: {
    name: 'Standard_D2ds_v4'
    tier: 'GeneralPurpose'
  }
  properties: {
    version: '8.0.21'
    administratorLogin: 'mysqladmin'
    storage: {
      storageSizeGB: 128
    }
  }

  resource secureTransport 'configurations@2023-06-30' = {
    name: 'require_secure_transport'
    properties: {
      value: 'OFF'
      source: 'user-override'
    }
  }
}
