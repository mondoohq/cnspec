resource "digitalocean_spaces_bucket" "data" {
  name   = "app-data"
  region = "nyc3"
  acl    = "private"
}

resource "digitalocean_spaces_key" "app" {
  name = "app-key"

  grant {
    bucket     = digitalocean_spaces_bucket.data.name
    permission = "readwrite"
  }
}
