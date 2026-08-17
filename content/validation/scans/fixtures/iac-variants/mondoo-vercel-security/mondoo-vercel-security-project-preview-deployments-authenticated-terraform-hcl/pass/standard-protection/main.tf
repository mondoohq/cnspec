resource "vercel_project" "storefront" {
  name      = "storefront"
  framework = "nextjs"

  vercel_authentication = {
    deployment_type = "standard_protection_new"
  }
}
