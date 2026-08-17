# The IP rules are scoped, but a system bypass covering the whole internet takes every request
# around the firewall regardless.
resource "vercel_project" "storefront" {
  name = "storefront"
}

resource "vercel_firewall_config" "storefront" {
  project_id = vercel_project.storefront.id

  ip_rules {
    rule {
      action   = "deny"
      ip       = "203.0.113.0/24"
      hostname = "*"
    }
  }
}

resource "vercel_firewall_bypass" "everyone" {
  project_id = vercel_project.storefront.id
  domain     = "storefront.example.com"
  source_ip  = "0.0.0.0/0"
}
