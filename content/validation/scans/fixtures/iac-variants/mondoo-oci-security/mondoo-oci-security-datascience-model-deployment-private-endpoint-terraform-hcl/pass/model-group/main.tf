# Compliant: a model-group deployment served through a private endpoint.
resource "oci_datascience_model_deployment" "example" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  project_id     = "ocid1.datascienceproject.oc1..aaaaaaaaexample"

  model_deployment_configuration_details {
    deployment_type = "MODEL_GROUP"

    infrastructure_configuration_details {
      infrastructure_type = "INSTANCE_POOL"

      instance_configuration {
        instance_shape_name = "VM.Standard.E4.Flex"
        private_endpoint_id = "ocid1.datascienceprivateendpoint.oc1..aaaaaaaaexample"
      }
    }
  }
}
