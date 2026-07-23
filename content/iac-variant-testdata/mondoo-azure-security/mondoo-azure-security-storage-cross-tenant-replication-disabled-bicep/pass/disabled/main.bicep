resource storageAccount 'Microsoft.Storage/storageAccounts@2023-01-01' = {
  name: 'stprodeastus001'
  location: 'eastus'
  sku: {
    name: 'Standard_LRS'
  }
  kind: 'StorageV2'
  properties: {
    minimumTlsVersion: 'TLS1_2'
    allowBlobPublicAccess: false
    allowCrossTenantReplication: false
  }
}
