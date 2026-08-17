# The restriction is named and turned off, which keeps every non-git deploy path
# open to production.
data "netlify_site" "app" {
  name = "app"
}

resource "netlify_site_build_settings" "app" {
  site_id           = data.netlify_site.app.id
  build_command     = "npm run build"
  publish_directory = "dist"
  production_branch = "main"

  prevent_non_git_prod_deploys = false
}
