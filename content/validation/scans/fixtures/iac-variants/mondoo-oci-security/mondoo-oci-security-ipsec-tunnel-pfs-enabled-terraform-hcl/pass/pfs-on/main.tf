# Each rekey derives fresh material, so recovering one key exposes one interval.
resource "oci_core_ipsec_connection_tunnel_management" "tunnel_1" {
  ipsec_id     = oci_core_ipsec.vpn.id
  tunnel_id    = data.oci_core_ipsec_connection_tunnels.vpn.ip_sec_connection_tunnels[0].id
  display_name = "tunnel-1"
  ike_version  = "V2"

  phase_two_details {
    is_custom_phase_two_config = true
    is_pfs_enabled             = true
    dh_group                   = "GROUP14"
  }
}
