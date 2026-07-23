resource subnet 'Microsoft.Network/virtualNetworks/subnets@2023-09-01' = {
  name: 'vnet-prod/subnet-legacy'
  properties: {
    addressPrefix: '10.0.2.0/24'
    defaultOutboundAccess: true
  }
}
