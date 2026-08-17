resource "vercel_project" "storefront" {
  name = "storefront"
}

resource "vercel_firewall_config" "storefront" {
  project_id = vercel_project.storefront.id

  ip_rules {
    rule {
      action   = "bypass"
      ip       = "198.51.100.14/32"
      hostname = "*"
      notes    = "synthetic monitoring"
    }

    rule {
      action   = "deny"
      ip       = "203.0.113.0/24"
      hostname = "*"
    }
  }
}

resource "vercel_firewall_bypass" "monitoring" {
  project_id = vercel_project.storefront.id
  domain     = "storefront.example.com"
  source_ip  = "198.51.100.14"
}
