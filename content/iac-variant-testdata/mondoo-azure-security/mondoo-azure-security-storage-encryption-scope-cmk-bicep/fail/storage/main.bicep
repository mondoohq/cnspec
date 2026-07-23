resource storageAccount 'Microsoft.Storage/storageAccounts@2023-01-01' = {
  name: 'stprodencrypt001'
  location: 'eastus'
  sku: {
    name: 'Standard_LRS'
  }
  kind: 'StorageV2'
  properties: {
    minimumTlsVersion: 'TLS1_2'
  }
}

resource encryptionScope 'Microsoft.Storage/storageAccounts/encryptionScopes@2023-01-01' = {
  parent: storageAccount
  name: 'msscope'
  properties: {
    source: 'Microsoft.Storage'
  }
}
