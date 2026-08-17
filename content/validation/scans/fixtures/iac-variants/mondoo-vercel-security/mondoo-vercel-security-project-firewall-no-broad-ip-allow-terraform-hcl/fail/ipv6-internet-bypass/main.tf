resource "vercel_project" "storefront" {
  name = "storefront"
}

resource "vercel_firewall_config" "storefront" {
  project_id = vercel_project.storefront.id

  ip_rules {
    rule {
      action   = "bypass"
      ip       = "::/0"
      hostname = "*"
    }
  }
}
