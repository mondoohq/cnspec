resource "digitalocean_spaces_bucket" "assets" {
  name   = "app-assets"
  region = "nyc3"
}

resource "digitalocean_cdn" "assets" {
  origin        = digitalocean_spaces_bucket.assets.bucket_domain_name
  custom_domain = "cdn.example.com"
}
