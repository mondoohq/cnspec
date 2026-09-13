resource "digitalocean_database_cluster" "opensearch" {
  name       = "legacy-opensearch"
  engine     = "opensearch"
  version    = "1"
  size       = "db-s-1vcpu-2gb"
  region     = "nyc1"
  node_count = 1
}
