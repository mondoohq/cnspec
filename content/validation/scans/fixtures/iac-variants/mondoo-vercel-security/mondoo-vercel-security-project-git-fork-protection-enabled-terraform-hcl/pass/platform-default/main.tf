# git_fork_protection defaults to true, so a project that never mentions it holds fork builds
# for authorization.
resource "vercel_project" "storefront" {
  name = "storefront"

  git_repository = {
    type = "github"
    repo = "example-org/storefront"
  }
}
