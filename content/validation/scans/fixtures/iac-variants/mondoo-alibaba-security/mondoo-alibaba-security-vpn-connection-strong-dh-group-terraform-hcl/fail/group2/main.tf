# Group 2 is a 1024-bit MODP prime shared by everyone who selects it, which is
# exactly the case precomputation attacks are cheapest against.
resource "alicloud_vpn_connection" "site" {
  vpn_gateway_id      = "vpn-example"
  customer_gateway_id = "cgw-example"
  local_subnet        = ["10.0.0.0/16"]
  remote_subnet       = ["192.168.0.0/16"]

  ike_config {
    ike_enc_alg = "aes256"
    ike_version = "ikev2"
    ike_pfs     = "group2"
  }

  ipsec_config {
    ipsec_enc_alg = "aes256"
    ipsec_pfs     = "group2"
  }
}
