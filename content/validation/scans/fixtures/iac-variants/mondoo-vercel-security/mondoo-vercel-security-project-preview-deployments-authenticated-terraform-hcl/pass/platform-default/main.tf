# vercel_authentication is optional and computed. Vercel turns Standard Protection on for new
# projects, so a project that never mentions the argument keeps its deployments behind a login.
resource "vercel_project" "storefront" {
  name      = "storefront"
  framework = "nextjs"
}
