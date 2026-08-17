resource "vercel_project" "storefront" {
  name = "storefront"

  trusted_ips = {
    deployment_type = "standard_protection_new"
    protection_mode = "trusted_ip_required"
    addresses       = [{ value = "203.0.113.0/24", note = "office" }]
  }
}
