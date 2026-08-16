# Non-compliant: config uses a credential-named key.
resource "oci_functions_application" "leaky" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  display_name   = "order-processor"
  subnet_ids     = ["ocid1.subnet.oc1.phx.examplesubnet.abcdefghijklmnop"]

  config = {
    LOG_LEVEL   = "info"
    DB_PASSWORD = "s3cr3tvalue"
  }
}
