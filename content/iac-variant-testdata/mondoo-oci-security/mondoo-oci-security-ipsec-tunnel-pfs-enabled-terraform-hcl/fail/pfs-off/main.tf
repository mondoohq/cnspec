# Without forward secrecy every rekey reuses the same long-term material, so one
# recovered key decrypts the whole history of the tunnel, including traffic recorded
# months before the key was obtained.
resource "oci_core_ipsec_connection_tunnel_management" "tunnel_1" {
  ipsec_id     = oci_core_ipsec.vpn.id
  tunnel_id    = data.oci_core_ipsec_connection_tunnels.vpn.ip_sec_connection_tunnels[0].id
  display_name = "tunnel-1"
  ike_version  = "V2"

  phase_two_details {
    is_custom_phase_two_config = true
    is_pfs_enabled             = false
    dh_group                   = "GROUP14"
  }
}
