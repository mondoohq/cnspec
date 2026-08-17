resource "vercel_project" "storefront" {
  name = "storefront"
}

resource "vercel_firewall_config" "storefront" {
  project_id = vercel_project.storefront.id

  ip_rules {
    rule {
      action   = "bypass"
      ip       = "0.0.0.0/0"
      hostname = "*"
      notes    = "temporary while we debug the login flow"
    }
  }
}
