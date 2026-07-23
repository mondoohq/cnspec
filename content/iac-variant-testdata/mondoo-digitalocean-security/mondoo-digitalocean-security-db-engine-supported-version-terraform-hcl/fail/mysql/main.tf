resource "digitalocean_database_cluster" "mysql" {
  name       = "legacy-mysql"
  engine     = "mysql"
  version    = "5.7"
  size       = "db-s-1vcpu-1gb"
  region     = "nyc1"
  node_count = 1
}
