resource "digitalocean_vpc" "db" {
  name   = "db-vpc"
  region = "nyc1"
}

resource "digitalocean_database_cluster" "pg" {
  name                 = "prod-pg"
  engine               = "pg"
  version              = "16"
  size                 = "db-s-1vcpu-1gb"
  region               = "nyc1"
  node_count           = 1
  private_network_uuid = digitalocean_vpc.db.id
}
