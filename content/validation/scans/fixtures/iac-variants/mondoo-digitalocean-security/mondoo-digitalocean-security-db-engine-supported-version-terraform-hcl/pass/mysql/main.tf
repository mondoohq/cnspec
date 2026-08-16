resource "digitalocean_database_cluster" "mysql" {
  name       = "prod-mysql"
  engine     = "mysql"
  version    = "8"
  size       = "db-s-1vcpu-1gb"
  region     = "nyc1"
  node_count = 1
}
