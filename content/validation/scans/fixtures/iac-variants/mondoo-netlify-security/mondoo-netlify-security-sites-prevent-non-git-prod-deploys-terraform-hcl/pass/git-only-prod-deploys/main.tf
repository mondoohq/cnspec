# Production deploys must arrive as a git push to the production branch, so a CLI
# or API publish cannot reach production.
data "netlify_site" "app" {
  name = "app"
}

resource "netlify_site_build_settings" "app" {
  site_id           = data.netlify_site.app.id
  build_command     = "npm run build"
  publish_directory = "dist"
  production_branch = "main"

  prevent_non_git_prod_deploys = true
}
