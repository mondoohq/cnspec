resource "digitalocean_database_cluster" "kafka" {
  name       = "prod-kafka"
  engine     = "kafka"
  version    = "4.3"
  size       = "db-s-2vcpu-2gb"
  region     = "nyc1"
  node_count = 3
}
