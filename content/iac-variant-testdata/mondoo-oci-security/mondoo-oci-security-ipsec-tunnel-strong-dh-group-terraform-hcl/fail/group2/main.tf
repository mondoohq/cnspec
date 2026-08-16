# Group 2 is a 1024-bit MODP prime shared by everyone who selects it, which is the
# cheapest possible precomputation target. AES-256 above it protects nothing the key
# exchange did not already give away.
resource "oci_core_ipsec_connection_tunnel_management" "tunnel_1" {
  ipsec_id     = oci_core_ipsec.vpn.id
  tunnel_id    = data.oci_core_ipsec_connection_tunnels.vpn.ip_sec_connection_tunnels[0].id
  display_name = "tunnel-1"
  ike_version  = "V2"

  phase_one_details {
    is_custom_phase_one_config  = true
    custom_dh_group             = "GROUP2"
    custom_encryption_algorithm = "AES_256_CBC"
  }

  phase_two_details {
    is_custom_phase_two_config = true
    dh_group                   = "GROUP2"
    is_pfs_enabled             = true
  }
}
