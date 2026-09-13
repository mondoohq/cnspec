resource "digitalocean_database_cluster" "kafka" {
  name       = "legacy-kafka"
  engine     = "kafka"
  version    = "3.8"
  size       = "db-s-2vcpu-2gb"
  region     = "nyc1"
  node_count = 3
}
