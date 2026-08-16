resource "digitalocean_certificate" "cert" {
  name    = "cdn-cert"
  type    = "lets_encrypt"
  domains = ["static.example.com"]
}

resource "digitalocean_spaces_bucket" "assets" {
  name   = "static-assets"
  region = "nyc3"
}

resource "digitalocean_cdn" "static" {
  origin         = digitalocean_spaces_bucket.assets.bucket_domain_name
  custom_domain  = "static.example.com"
  certificate_id = digitalocean_certificate.cert.id
}
