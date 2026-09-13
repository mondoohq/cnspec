resource "digitalocean_database_cluster" "mysql" {
  name       = "legacy-mysql-8"
  engine     = "mysql"
  version    = "8"
  size       = "db-s-1vcpu-1gb"
  region     = "nyc1"
  node_count = 1
}
