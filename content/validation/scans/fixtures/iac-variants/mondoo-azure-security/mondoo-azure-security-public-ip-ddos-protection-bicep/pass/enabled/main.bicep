resource publicIp 'Microsoft.Network/publicIPAddresses@2023-09-01' = {
  name: 'pip-gateway'
  location: 'eastus'
  sku: {
    name: 'Standard'
    tier: 'Regional'
  }
  properties: {
    publicIPAllocationMethod: 'Static'
    publicIPAddressVersion: 'IPv4'
    ddosSettings: {
      protectionMode: 'Enabled'
    }
  }
}
