# An unlocked registration can be transferred away without the extra
# confirmation step, which is how domain hijacks usually start.
resource "google_clouddomains_registration" "corp" {
  project     = var.project_id
  location    = "global"
  domain_name = "example-corp.com"

  yearly_price {
    currency_code = "USD"
    units         = 12
  }

  management_settings {
    transfer_lock_state = "UNLOCKED"
  }
}
