resource "digitalocean_spaces_bucket" "assets" {
  name   = "public-assets"
  region = "nyc3"
}

resource "digitalocean_cdn" "public" {
  origin = digitalocean_spaces_bucket.assets.bucket_domain_name
}
