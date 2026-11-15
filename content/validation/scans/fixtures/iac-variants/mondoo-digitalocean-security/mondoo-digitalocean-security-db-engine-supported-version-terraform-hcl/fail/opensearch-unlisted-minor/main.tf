# OpenSearch 2.17 is no longer in DigitalOcean's supported list; only 2.19 is.
resource "digitalocean_database_cluster" "search" {
  name       = "example-search"
  engine     = "opensearch"
  version    = "2.17"
  size       = "db-s-1vcpu-2gb"
  region     = "nyc1"
  node_count = 1
}
