resource "digitalocean_certificate" "cert" {
  name    = "cdn-cert"
  type    = "lets_encrypt"
  domains = ["cdn.example.com"]
}

resource "digitalocean_spaces_bucket" "assets" {
  name   = "app-assets"
  region = "nyc3"
}

resource "digitalocean_cdn" "assets" {
  origin           = digitalocean_spaces_bucket.assets.bucket_domain_name
  custom_domain    = "cdn.example.com"
  certificate_name = digitalocean_certificate.cert.name
}
