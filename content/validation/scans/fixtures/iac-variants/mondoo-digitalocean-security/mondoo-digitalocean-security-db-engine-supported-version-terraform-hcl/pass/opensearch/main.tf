resource "digitalocean_database_cluster" "opensearch" {
  name       = "prod-opensearch"
  engine     = "opensearch"
  version    = "2.19"
  size       = "db-s-1vcpu-2gb"
  region     = "nyc1"
  node_count = 1
}
