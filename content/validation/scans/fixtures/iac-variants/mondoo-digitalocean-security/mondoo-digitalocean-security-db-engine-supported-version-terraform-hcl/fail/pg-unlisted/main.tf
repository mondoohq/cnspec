# A version with no entry in the lifecycle table fails rather than passing unexamined.
resource "digitalocean_database_cluster" "pg" {
  name       = "next-pg"
  engine     = "pg"
  version    = "99"
  size       = "db-s-1vcpu-1gb"
  region     = "nyc1"
  node_count = 1
}
