resource "google_compute_vpn_tunnel" "fail" {
  name          = "tunnel-1"
  peer_ip       = "15.0.0.120"
  shared_secret = "a secret message"
  ike_version   = 1
  target_vpn_gateway = google_compute_vpn_gateway.target_gateway.id
}
