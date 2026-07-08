resource sa 'Microsoft.Storage/storageAccounts@2023-01-01' = {
  name: 'securestorageacct'
  location: 'eastus'
  sku: {
    name: 'Standard_LRS'
  }
  kind: 'StorageV2'
  properties: {
    minimumTlsVersion: 'TLS1_2'
    allowBlobPublicAccess: false
    networkAcls: {
      defaultAction: 'Deny'
      bypass: 'AzureServices'
      ipRules: [
        {
          value: '203.0.113.0/24'
        }
      ]
    }
  }
}
