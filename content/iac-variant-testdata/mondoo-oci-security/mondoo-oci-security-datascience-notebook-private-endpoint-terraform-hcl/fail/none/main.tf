# Non-compliant: notebook session has no private endpoint configured.
resource "oci_datascience_notebook_session" "example" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  project_id     = "ocid1.datascienceproject.oc1..aaaaaaaaexample"

  notebook_session_config_details {
    shape     = "VM.Standard2.1"
    subnet_id = "ocid1.subnet.oc1..aaaaaaaaexamplesubnet"

    notebook_session_shape_config_details {
      ocpus         = 1
      memory_in_gbs = 16
    }
  }
}
