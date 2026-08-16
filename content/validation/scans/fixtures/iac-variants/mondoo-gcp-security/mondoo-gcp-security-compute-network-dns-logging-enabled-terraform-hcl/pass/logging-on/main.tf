# Compliant: DNS policy with query logging enabled.
resource "google_dns_policy" "policy" {
  name           = "network-dns-policy"
  enable_logging = true

  networks {
    network_url = google_compute_network.vpc.id
  }
}
