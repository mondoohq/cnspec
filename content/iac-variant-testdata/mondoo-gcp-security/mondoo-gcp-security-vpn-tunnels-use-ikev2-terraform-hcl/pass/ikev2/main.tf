resource "google_compute_vpn_tunnel" "pass" {
  name          = "tunnel-1"
  peer_ip       = "15.0.0.120"
  shared_secret = "a secret message"
  ike_version   = 2
  target_vpn_gateway = google_compute_vpn_gateway.target_gateway.id
}
