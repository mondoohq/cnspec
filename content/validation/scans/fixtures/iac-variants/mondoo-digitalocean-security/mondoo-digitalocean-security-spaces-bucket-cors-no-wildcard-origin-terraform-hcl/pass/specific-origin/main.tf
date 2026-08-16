resource "digitalocean_spaces_bucket" "assets" {
  name   = "app-assets"
  region = "nyc3"
  acl    = "private"

  cors_rule {
    allowed_headers = ["*"]
    allowed_methods = ["GET"]
    allowed_origins = ["https://www.example.com"]
    max_age_seconds = 3000
  }
}
