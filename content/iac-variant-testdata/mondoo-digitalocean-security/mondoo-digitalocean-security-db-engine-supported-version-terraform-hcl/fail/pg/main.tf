resource "digitalocean_database_cluster" "pg" {
  name       = "legacy-pg"
  engine     = "pg"
  version    = "13"
  size       = "db-s-1vcpu-1gb"
  region     = "nyc1"
  node_count = 1
}
