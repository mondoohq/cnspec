resource storageAccount 'Microsoft.Storage/storageAccounts@2023-01-01' = {
  name: 'stprodeastus002'
  location: 'eastus'
  sku: {
    name: 'Standard_LRS'
  }
  kind: 'StorageV2'
  properties: {
    minimumTlsVersion: 'TLS1_2'
    isHnsEnabled: true
    isSftpEnabled: false
  }
}
