resource vnet 'Microsoft.Network/virtualNetworks@2023-09-01' = {
  name: 'vnet-test-eastus'
  location: 'eastus'
  properties: {
    addressSpace: {
      addressPrefixes: [
        '10.2.0.0/16'
      ]
    }
  }
}
