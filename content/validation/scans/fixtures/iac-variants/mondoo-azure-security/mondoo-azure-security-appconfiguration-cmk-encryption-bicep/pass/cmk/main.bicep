resource configStore 'Microsoft.AppConfiguration/configurationStores@2023-03-01' = {
  name: 'example-appconfig'
  location: 'eastus'
  sku: {
    name: 'Standard'
  }
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    encryption: {
      keyVaultProperties: {
        keyIdentifier: 'https://example-kv.vault.azure.net/keys/appconfig-cmk'
        identityClientId: 'aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee'
      }
    }
    publicNetworkAccess: 'Disabled'
  }
}
