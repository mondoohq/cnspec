# Neon allows public connections by default, so omitting the argument leaves the
# project reachable from the internet.
resource "neon_project" "app" {
  name = "app"
}
