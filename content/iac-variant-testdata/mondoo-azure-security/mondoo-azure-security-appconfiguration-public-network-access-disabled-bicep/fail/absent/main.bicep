resource configStore 'Microsoft.AppConfiguration/configurationStores@2023-03-01' = {
  name: 'example-appconfig'
  location: 'eastus'
  sku: {
    name: 'Standard'
  }
  properties: {
    disableLocalAuth: true
  }
}
