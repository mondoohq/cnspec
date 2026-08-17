# Netlify leaves the restriction off by default, so omitting the argument leaves
# the site accepting a CLI, API, or drag-and-drop publish straight to production.
data "netlify_site" "app" {
  name = "app"
}

resource "netlify_site_build_settings" "app" {
  site_id           = data.netlify_site.app.id
  build_command     = "npm run build"
  publish_directory = "dist"
  production_branch = "main"
}
