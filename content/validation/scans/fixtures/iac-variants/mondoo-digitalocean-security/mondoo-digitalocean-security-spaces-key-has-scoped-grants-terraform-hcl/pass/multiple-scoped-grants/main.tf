resource "digitalocean_spaces_bucket" "logs" {
  name   = "app-logs"
  region = "nyc3"
  acl    = "private"
}

resource "digitalocean_spaces_bucket" "uploads" {
  name   = "app-uploads"
  region = "nyc3"
  acl    = "private"
}

resource "digitalocean_spaces_key" "app" {
  name = "app-key"

  grant {
    bucket     = digitalocean_spaces_bucket.logs.name
    permission = "read"
  }

  grant {
    bucket     = digitalocean_spaces_bucket.uploads.name
    permission = "readwrite"
  }
}
