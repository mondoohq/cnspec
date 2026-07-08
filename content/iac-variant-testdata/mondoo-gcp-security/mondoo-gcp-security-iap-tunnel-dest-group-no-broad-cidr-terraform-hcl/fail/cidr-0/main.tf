# Non-compliant: 0.0.0.0/0 exposes the entire internet as a destination.
resource "google_iap_tunnel_dest_group" "fail_example" {
  region     = "us-central1"
  group_name = "any-group"
  cidrs = [
    "10.4.0.0/24",
    "0.0.0.0/0",
  ]
}
