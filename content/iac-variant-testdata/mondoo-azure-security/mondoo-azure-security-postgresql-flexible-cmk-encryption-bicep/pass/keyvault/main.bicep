resource pgIdentity 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' = {
  name: 'contoso-postgres-identity'
  location: 'eastus'
}

resource postgresServer 'Microsoft.DBforPostgreSQL/flexibleServers@2023-06-01-preview' = {
  name: 'contoso-postgres'
  location: 'eastus'
  identity: {
    type: 'UserAssigned'
    userAssignedIdentities: {
      '${pgIdentity.id}': {}
    }
  }
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
    dataEncryption: {
      type: 'AzureKeyVault'
      primaryKeyURI: 'https://contoso-kv.vault.azure.net/keys/pg-cmk/abcdef1234567890'
      primaryUserAssignedIdentityId: pgIdentity.id
    }
  }
}
