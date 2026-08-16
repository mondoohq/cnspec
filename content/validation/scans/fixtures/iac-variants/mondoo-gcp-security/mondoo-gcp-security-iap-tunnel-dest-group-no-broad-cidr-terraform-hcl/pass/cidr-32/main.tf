# Compliant: destination CIDRs target single hosts (/32).
resource "google_iap_tunnel_dest_group" "pass_example" {
  region     = "us-central1"
  group_name = "bastion-group"
  cidrs = [
    "10.0.5.7/32",
  ]
}
