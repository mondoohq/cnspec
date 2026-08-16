resource "digitalocean_spaces_bucket" "data" {
  name   = "app-data"
  region = "nyc3"
  acl    = "private"
}
