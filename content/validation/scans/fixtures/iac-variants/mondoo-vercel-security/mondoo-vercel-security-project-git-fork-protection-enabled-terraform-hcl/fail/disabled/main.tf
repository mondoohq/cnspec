resource "vercel_project" "storefront" {
  name                = "storefront"
  git_fork_protection = false

  git_repository = {
    type = "github"
    repo = "example-org/storefront"
  }
}
