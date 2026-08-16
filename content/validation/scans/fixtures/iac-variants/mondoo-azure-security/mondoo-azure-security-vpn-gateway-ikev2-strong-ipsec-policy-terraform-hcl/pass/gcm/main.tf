resource "azurerm_virtual_network_gateway_connection" "example" {
  name                = "example-connection"
  resource_group_name = "example-rg"
  location            = "eastus"

  type                       = "IPsec"
  virtual_network_gateway_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.Network/virtualNetworkGateways/example-vpn-gw"
  local_network_gateway_id   = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.Network/localNetworkGateways/example-lng"
  shared_key                 = "S3cr3tSharedKey!"

  ipsec_policy {
    dh_group         = "DHGroup14"
    ike_encryption   = "GCMAES256"
    ike_integrity    = "SHA256"
    ipsec_encryption = "GCMAES256"
    ipsec_integrity  = "SHA256"
    pfs_group        = "PFS2048"
    sa_datasize      = 102400000
    sa_lifetime      = 27000
  }
}
