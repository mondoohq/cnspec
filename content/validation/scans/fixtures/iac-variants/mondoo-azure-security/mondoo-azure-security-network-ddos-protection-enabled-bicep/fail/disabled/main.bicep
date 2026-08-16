resource vnet 'Microsoft.Network/virtualNetworks@2023-09-01' = {
  name: 'vnet-dev-001'
  location: 'eastus'
  properties: {
    addressSpace: {
      addressPrefixes: [
        '10.10.0.0/16'
      ]
    }
    enableDdosProtection: false
    subnets: [
      {
        name: 'default'
        properties: {
          addressPrefix: '10.10.0.0/24'
        }
      }
    ]
  }
}
