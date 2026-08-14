# Omitting management_settings entirely leaves the lock state unmanaged.
resource "google_clouddomains_registration" "corp" {
  project     = var.project_id
  location    = "global"
  domain_name = "example-corp.com"

  yearly_price {
    currency_code = "USD"
    units         = 12
  }
}
