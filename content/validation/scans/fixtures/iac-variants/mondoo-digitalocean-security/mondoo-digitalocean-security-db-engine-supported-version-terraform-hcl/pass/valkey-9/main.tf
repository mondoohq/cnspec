resource "digitalocean_database_cluster" "cache" {
  name       = "prod-valkey"
  engine     = "valkey"
  version    = "9"
  size       = "db-s-1vcpu-1gb"
  region     = "nyc1"
  node_count = 1
}
