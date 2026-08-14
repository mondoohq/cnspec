resource "oci_core_ipsec" "vpn" {
  compartment_id = var.compartment_ocid
  cpe_id         = oci_core_cpe.on_prem.id
  drg_id         = oci_core_drg.main.id
  display_name   = "prod-vpn"

  static_routes = ["10.10.0.0/16"]
}

resource "oci_core_ipsec_connection_tunnel_management" "tunnel_1" {
  ipsec_id     = oci_core_ipsec.vpn.id
  tunnel_id    = data.oci_core_ipsec_connection_tunnels.vpn.ip_sec_connection_tunnels[0].id
  display_name = "tunnel-1"
  routing      = "BGP"
  ike_version  = "V2"

  bgp_session_info {
    customer_bgp_asn      = "64512"
    customer_interface_ip = "10.0.0.16/31"
    oracle_interface_ip   = "10.0.0.17/31"
  }
}
