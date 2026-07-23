# Non-compliant: enable_logging not set (defaults to disabled).
resource "google_dns_policy" "policy" {
  name = "network-dns-policy"

  networks {
    network_url = google_compute_network.vpc.id
  }
}
