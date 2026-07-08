resource "digitalocean_spaces_bucket" "assets" {
  name   = "app-assets"
  region = "nyc3"
  acl    = "private"
}
