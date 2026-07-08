resource "digitalocean_database_cluster" "pg" {
  name       = "prod-pg"
  engine     = "pg"
  version    = "16"
  size       = "db-s-1vcpu-1gb"
  region     = "nyc1"
  node_count = 1
}
