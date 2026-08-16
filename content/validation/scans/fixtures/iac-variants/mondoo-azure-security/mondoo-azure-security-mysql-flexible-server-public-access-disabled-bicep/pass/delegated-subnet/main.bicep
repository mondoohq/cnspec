resource mysql 'Microsoft.DBforMySQL/flexibleServers@2023-12-30' = {
  name: 'contoso-mysql-vnet'
  location: 'eastus'
  sku: {
    name: 'Standard_D2ds_v4'
    tier: 'GeneralPurpose'
  }
  properties: {
    administratorLogin: 'dbadmin'
    version: '8.0.21'
    network: {
      delegatedSubnetResourceId: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-net/providers/Microsoft.Network/virtualNetworks/vnet-db/subnets/mysql-subnet'
      privateDnsZoneResourceId: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-net/providers/Microsoft.Network/privateDnsZones/contoso.mysql.database.azure.com'
    }
  }
}
