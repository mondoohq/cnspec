resource "vercel_project" "storefront" {
  name = "storefront"

  vercel_authentication = {
    deployment_type = "standard_protection_new"
  }
}
