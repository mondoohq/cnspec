resource "oci_core_ipsec" "vpn" {
  compartment_id = var.compartment_ocid
  cpe_id         = oci_core_cpe.on_prem.id
  drg_id         = oci_core_drg.main.id
  display_name   = "legacy-vpn"

  static_routes = ["10.10.0.0/16"]
}

# IKEv1 is deprecated and offers weaker key exchange than IKEv2.
resource "oci_core_ipsec_connection_tunnel_management" "tunnel_1" {
  ipsec_id     = oci_core_ipsec.vpn.id
  tunnel_id    = data.oci_core_ipsec_connection_tunnels.vpn.ip_sec_connection_tunnels[0].id
  display_name = "tunnel-1"
  routing      = "STATIC"
  ike_version  = "V1"
}
