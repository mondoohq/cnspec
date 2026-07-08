# Non-compliant: /8 is an overly broad destination range.
resource "google_iap_tunnel_dest_group" "fail_example" {
  region     = "us-central1"
  group_name = "broad-group"
  cidrs = [
    "10.0.0.0/8",
  ]
}
