resource subnet 'Microsoft.Network/virtualNetworks/subnets@2023-09-01' = {
  name: 'vnet-prod/subnet-app'
  properties: {
    addressPrefix: '10.0.1.0/24'
    defaultOutboundAccess: false
  }
}
