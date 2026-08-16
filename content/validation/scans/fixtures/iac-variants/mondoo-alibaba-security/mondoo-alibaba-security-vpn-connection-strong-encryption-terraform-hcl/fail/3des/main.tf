# 3DES has a 64-bit block, which makes a long-lived high-volume site-to-site
# tunnel vulnerable to birthday attacks on the ciphertext.
resource "alicloud_vpn_connection" "site" {
  vpn_gateway_id      = "vpn-example"
  customer_gateway_id = "cgw-example"
  local_subnet        = ["10.0.0.0/16"]
  remote_subnet       = ["192.168.0.0/16"]

  ike_config {
    ike_enc_alg = "3des"
    ike_version = "ikev2"
    ike_pfs     = "group14"
  }

  ipsec_config {
    ipsec_enc_alg = "3des"
    ipsec_pfs     = "group14"
  }
}
