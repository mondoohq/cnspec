resource sqlServer 'Microsoft.Sql/servers@2023-05-01-preview' = {
  name: 'sql-prod-eastus'
  location: 'eastus'
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    administratorLogin: 'sqladmin'
    version: '12.0'
  }
}

resource tdeProtector 'Microsoft.Sql/servers/encryptionProtector@2023-05-01-preview' = {
  parent: sqlServer
  name: 'current'
  properties: {
    serverKeyType: 'AzureKeyVault'
    serverKeyName: 'contosokv_tde-key_78bd76f0e5c74c7bb3e3f7f4e0b1a2c3'
    autoRotationEnabled: true
  }
}
