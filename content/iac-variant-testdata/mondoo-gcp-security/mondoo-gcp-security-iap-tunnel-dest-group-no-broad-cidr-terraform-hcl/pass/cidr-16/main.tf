# Compliant: destination CIDRs are narrowly scoped (/16 and /24).
resource "google_iap_tunnel_dest_group" "pass_example" {
  region = "us-central1"
  group_name = "prod-db-group"
  cidrs = [
    "10.1.0.0/16",
    "10.2.3.0/24",
  ]
}
