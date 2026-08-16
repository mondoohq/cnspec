# Neon stores role passwords by default, so omitting the argument retains them.
resource "neon_project" "app" {
  name = "app"
}
