# Compliant: endpoint is bound to a Generative AI private endpoint.
resource "oci_generative_ai_endpoint" "example" {
  compartment_id                    = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  dedicated_ai_cluster_id           = "ocid1.generativeaidedicatedaicluster.oc1..aaaaaaaaexample"
  model_id                          = "ocid1.generativeaimodel.oc1..aaaaaaaaexample"
  generative_ai_private_endpoint_id = "ocid1.generativeaiprivateendpoint.oc1..aaaaaaaaexample"
}
