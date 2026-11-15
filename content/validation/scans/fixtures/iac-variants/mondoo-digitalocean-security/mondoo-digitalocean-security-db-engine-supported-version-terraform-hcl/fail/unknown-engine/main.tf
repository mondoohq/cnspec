# An engine with no entry in the lifecycle table fails rather than passing unexamined.
resource "digitalocean_database_cluster" "example" {
  name       = "example-db"
  engine     = "cassandra"
  version    = "5"
  size       = "db-s-1vcpu-1gb"
  region     = "nyc1"
  node_count = 1
}
