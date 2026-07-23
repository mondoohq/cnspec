# Non-compliant: config value contains an AWS access key ID.
resource "oci_functions_application" "leaky" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  display_name   = "s3-syncer"
  subnet_ids     = ["ocid1.subnet.oc1.phx.examplesubnet.abcdefghijklmnop"]

  config = {
    LOG_LEVEL      = "info"
    UPSTREAM_TOKEN = "AKIAIOSFODNN7EXAMPLE"
  }
}
