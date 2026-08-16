# No ike_config and no ipsec_config, so the connection takes Alibaba's defaults. The
# documented default for both ike_pfs and ipsec_pfs is group2, which is one of the weak
# 1024-bit MODP groups, so an omitted block is a weak configuration and not a neutral one.
resource "alicloud_vpn_connection" "site" {
  vpn_gateway_id      = "vpn-example"
  customer_gateway_id = "cgw-example"
  local_subnet        = ["10.0.0.0/16"]
  remote_subnet       = ["192.168.0.0/16"]
}
