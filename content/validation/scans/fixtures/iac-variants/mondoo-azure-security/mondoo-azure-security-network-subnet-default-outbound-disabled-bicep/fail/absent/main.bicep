resource subnet 'Microsoft.Network/virtualNetworks/subnets@2023-09-01' = {
  name: 'vnet-prod/subnet-web'
  properties: {
    addressPrefix: '10.0.3.0/24'
  }
}
