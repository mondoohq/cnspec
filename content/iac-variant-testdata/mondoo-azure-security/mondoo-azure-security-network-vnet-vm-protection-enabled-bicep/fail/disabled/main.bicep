resource vnet 'Microsoft.Network/virtualNetworks@2023-09-01' = {
  name: 'vnet-dev-eastus'
  location: 'eastus'
  properties: {
    addressSpace: {
      addressPrefixes: [
        '10.1.0.0/16'
      ]
    }
    enableVmProtection: false
  }
}
