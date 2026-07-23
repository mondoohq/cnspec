resource mysql 'Microsoft.DBforMySQL/flexibleServers@2023-12-30' = {
  name: 'contoso-mysql'
  location: 'eastus'
  sku: {
    name: 'Standard_D2ds_v4'
    tier: 'GeneralPurpose'
  }
  identity: {
    type: 'UserAssigned'
    userAssignedIdentities: {
      '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-db/providers/Microsoft.ManagedIdentity/userAssignedIdentities/mysql-identity': {}
    }
  }
  properties: {
    administratorLogin: 'dbadmin'
    version: '8.0.21'
  }
}
