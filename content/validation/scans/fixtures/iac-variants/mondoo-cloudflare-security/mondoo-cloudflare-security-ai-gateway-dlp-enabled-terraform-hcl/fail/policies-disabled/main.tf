resource "cloudflare_ai_gateway" "example" {
  account_id                 = "0123456789abcdef0123456789abcdef"
  id                         = "production"
  authentication             = true
  cache_invalidate_on_update = true
  cache_ttl                  = 0
  collect_logs               = true
  rate_limiting_interval     = 60
  rate_limiting_limit        = 100

  dlp = {
    enabled = true
    policies = [{
      id       = "block-sensitive"
      enabled  = false
      action   = "BLOCK"
      check    = ["REQUEST", "RESPONSE"]
      profiles = ["00000000-0000-0000-0000-000000000000"]
    }]
  }
}
