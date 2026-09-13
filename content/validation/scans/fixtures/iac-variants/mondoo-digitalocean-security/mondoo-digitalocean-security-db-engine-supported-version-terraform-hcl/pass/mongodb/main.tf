resource "digitalocean_database_cluster" "mongodb" {
  name       = "prod-mongodb"
  engine     = "mongodb"
  version    = "8.0"
  size       = "gd-2vcpu-8gb"
  region     = "nyc1"
  node_count = 1
}
