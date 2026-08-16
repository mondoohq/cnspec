resource storageAccount 'Microsoft.Storage/storageAccounts@2023-01-01' = {
  name: 'examplestorageacct'
  location: 'eastus'
  sku: {
    name: 'Standard_LRS'
  }
  kind: 'StorageV2'
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    accessTier: 'Hot'
    minimumTlsVersion: 'TLS1_2'
    supportsHttpsTrafficOnly: true
    encryption: {
      keySource: 'Microsoft.Keyvault'
      keyVaultProperties: {
        keyname: 'storagekey'
        keyvaulturi: 'https://examplevault.vault.azure.net'
      }
      services: {
        blob: {
          enabled: true
        }
        file: {
          enabled: true
        }
      }
    }
  }
}
