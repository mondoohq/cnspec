resource "cloudflare_r2_managed_domain" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  bucket_name = "example-bucket"
  enabled     = false
}
