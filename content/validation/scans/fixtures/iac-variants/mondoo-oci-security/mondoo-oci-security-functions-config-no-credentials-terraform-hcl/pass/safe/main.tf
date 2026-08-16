# Compliant: config holds only non-secret runtime settings.
resource "oci_functions_application" "compliant" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  display_name   = "order-processor"
  subnet_ids     = ["ocid1.subnet.oc1.phx.examplesubnet.abcdefghijklmnop"]

  config = {
    LOG_LEVEL      = "info"
    MAX_BATCH_SIZE = "100"
    REGION         = "us-phoenix-1"
  }
}
