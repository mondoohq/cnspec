# A firewall with only managed rulesets configures no IP rules at all, so nothing is let past.
resource "vercel_project" "storefront" {
  name = "storefront"
}

resource "vercel_firewall_config" "storefront" {
  project_id = vercel_project.storefront.id

  managed_rulesets {
    bot_protection {
      active = true
      action = "deny"
    }
  }
}
