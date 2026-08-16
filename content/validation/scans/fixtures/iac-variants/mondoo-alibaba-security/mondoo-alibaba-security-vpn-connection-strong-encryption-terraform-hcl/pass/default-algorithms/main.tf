# No ike_config and no ipsec_config, so the connection takes Alibaba's defaults. The
# documented default for both encryption algorithms is aes, so an omitted block is not a
# weak cipher selection. The Diffie-Hellman check treats the same omission as a failure,
# because its default is group2.
resource "alicloud_vpn_connection" "site" {
  vpn_gateway_id      = "vpn-example"
  customer_gateway_id = "cgw-example"
  local_subnet        = ["10.0.0.0/16"]
  remote_subnet       = ["192.168.0.0/16"]
}
