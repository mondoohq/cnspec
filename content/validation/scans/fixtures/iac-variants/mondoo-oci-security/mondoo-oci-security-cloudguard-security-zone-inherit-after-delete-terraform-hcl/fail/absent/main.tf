# Non-compliant: is_inheritance_after_delete_enabled is not set, so it defaults
# to disabled.
resource "oci_cloud_guard_security_zone" "example" {
  compartment_id          = var.compartment_id
  display_name            = "prod-security-zone"
  security_zone_recipe_id = oci_cloud_guard_security_recipe.example.id
}
