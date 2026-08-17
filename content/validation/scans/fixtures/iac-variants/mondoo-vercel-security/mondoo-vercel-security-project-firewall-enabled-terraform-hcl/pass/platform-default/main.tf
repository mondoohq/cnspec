# enabled defaults to true on vercel_firewall_config, so declaring rules without touching the
# toggle leaves the firewall on.
resource "vercel_project" "storefront" {
  name = "storefront"
}

resource "vercel_firewall_config" "storefront" {
  project_id = vercel_project.storefront.id

  ip_rules {
    rule {
      action   = "deny"
      ip       = "198.51.100.0/24"
      hostname = "*"
    }
  }
}
