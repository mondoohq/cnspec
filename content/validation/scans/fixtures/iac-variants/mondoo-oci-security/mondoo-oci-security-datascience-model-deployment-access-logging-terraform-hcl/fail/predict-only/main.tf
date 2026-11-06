# Non-compliant: only predict logs are configured.
resource "oci_datascience_model_deployment" "example" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  project_id     = "ocid1.datascienceproject.oc1..aaaaaaaaexample"

  category_log_details {
    predict {
      log_group_id = "ocid1.loggroup.oc1..aaaaaaaaexample"
      log_id       = "ocid1.log.oc1..aaaaaaaaexample"
    }
  }

  model_deployment_configuration_details {
    deployment_type = "SINGLE_MODEL"
    model_configuration_details {
      model_id = "ocid1.datasciencemodel.oc1..aaaaaaaaexample"
      instance_configuration {
        instance_shape_name = "VM.Standard.E4.Flex"
      }
    }
  }
}
