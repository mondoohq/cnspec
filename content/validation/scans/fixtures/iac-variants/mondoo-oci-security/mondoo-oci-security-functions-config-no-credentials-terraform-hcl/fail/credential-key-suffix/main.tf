# Non-compliant: credential-named keys that carry a service prefix.
resource "oci_functions_application" "leaky" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  display_name   = "order-processor"
  subnet_ids     = ["ocid1.subnet.oc1.phx.examplesubnet.abcdefghijklmnop"]

  config = {
    POSTGRES_PASSWORD = "s3cr3tvalue"
    GITHUB_TOKEN      = "placeholder"
  }
}
