resource "digitalocean_database_cluster" "cache" {
  name       = "prod-redis"
  engine     = "redis"
  version    = "7"
  size       = "db-s-1vcpu-1gb"
  region     = "nyc1"
  node_count = 1
}
