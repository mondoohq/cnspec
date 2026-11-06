# Compliant: personal information is screened on input and output.
resource "oci_generative_ai_agent_agent_endpoint" "example" {
  agent_id       = "ocid1.genaiagent.oc1..aaaaaaaaexample"
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"

  guardrail_config {
    personally_identifiable_information_config {
      input_guardrail_mode  = "BLOCK"
      output_guardrail_mode = "INFORM"
    }
  }
}
