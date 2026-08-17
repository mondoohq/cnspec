# password_protection is optional and not computed, so Vercel leaves it off until it is declared.
resource "vercel_project" "storefront" {
  name = "storefront"
}
