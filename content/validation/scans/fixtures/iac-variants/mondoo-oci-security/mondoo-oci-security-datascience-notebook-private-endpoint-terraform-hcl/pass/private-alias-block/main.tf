# Compliant: the session uses the aliased configuration block and binds a private endpoint.
resource "oci_datascience_notebook_session" "example" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  project_id     = "ocid1.datascienceproject.oc1..aaaaaaaaexample"

  notebook_session_configuration_details {
    shape               = "VM.Standard2.1"
    subnet_id           = "ocid1.subnet.oc1..aaaaaaaaexamplesubnet"
    private_endpoint_id = "ocid1.datascienceprivateendpoint.oc1..aaaaaaaaexample"
  }
}
