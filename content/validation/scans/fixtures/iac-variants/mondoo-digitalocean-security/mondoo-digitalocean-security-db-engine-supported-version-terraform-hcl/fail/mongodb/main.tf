resource "digitalocean_database_cluster" "mongodb" {
  name       = "legacy-mongodb"
  engine     = "mongodb"
  version    = "6.0"
  size       = "gd-2vcpu-8gb"
  region     = "nyc1"
  node_count = 1
}
