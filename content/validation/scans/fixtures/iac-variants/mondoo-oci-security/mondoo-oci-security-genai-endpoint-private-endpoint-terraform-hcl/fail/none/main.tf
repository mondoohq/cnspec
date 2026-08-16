# Non-compliant: no private endpoint bound, so traffic uses public paths.
resource "oci_generative_ai_endpoint" "example" {
  compartment_id          = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  dedicated_ai_cluster_id = "ocid1.generativeaidedicatedaicluster.oc1..aaaaaaaaexample"
  model_id                = "ocid1.generativeaimodel.oc1..aaaaaaaaexample"
}
