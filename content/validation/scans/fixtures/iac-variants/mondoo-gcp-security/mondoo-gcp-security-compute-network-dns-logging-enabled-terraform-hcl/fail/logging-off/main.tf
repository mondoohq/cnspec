# Non-compliant: DNS policy with query logging disabled.
resource "google_dns_policy" "policy" {
  name           = "network-dns-policy"
  enable_logging = false

  networks {
    network_url = google_compute_network.vpc.id
  }
}
