@description('Name of the Data Factory')
param factoryName string = 'contoso-adf-prod'

@description('Deployment location')
param location string = resourceGroup().location

resource dataFactory 'Microsoft.DataFactory/factories@2018-06-01' = {
  name: factoryName
  location: location
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    encryption: {
      keyName: 'adf-cmk'
      keyVersion: '78bd76f0e5c74c7bb3e3f7f4e0b1a2c3'
      vaultBaseUrl: 'https://contoso-kv.vault.azure.net/'
    }
  }
}
