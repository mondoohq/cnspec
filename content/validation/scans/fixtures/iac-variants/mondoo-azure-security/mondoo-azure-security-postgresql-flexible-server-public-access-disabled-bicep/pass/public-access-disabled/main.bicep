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
      publicNetworkAccess: 'Disabled'
      delegatedSubnetResourceId: subnet.id
      privateDnsZoneArmResourceId: privateDnsZone.id
    }
    storage: {
      storageSizeGB: 128
    }
  }
}

resource subnet 'Microsoft.Network/virtualNetworks/subnets@2023-05-01' existing = {
  name: 'vnet/postgres-subnet'
}

resource privateDnsZone 'Microsoft.Network/privateDnsZones@2020-06-01' existing = {
  name: 'contoso.postgres.database.azure.com'
}
