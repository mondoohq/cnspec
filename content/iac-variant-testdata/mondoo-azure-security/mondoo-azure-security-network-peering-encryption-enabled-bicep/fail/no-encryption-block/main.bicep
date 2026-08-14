resource vnet 'Microsoft.Network/virtualNetworks@2023-09-01' = {
  name: 'prod-vnet'
  location: 'eastus'
  properties: {
    addressSpace: {
      addressPrefixes: [
        '10.0.0.0/16'
      ]
    }
  }
}
