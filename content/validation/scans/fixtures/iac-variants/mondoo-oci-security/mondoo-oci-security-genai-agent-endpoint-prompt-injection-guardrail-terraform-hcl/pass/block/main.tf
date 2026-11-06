# Compliant: the prompt injection guardrail blocks.
resource "oci_generative_ai_agent_agent_endpoint" "example" {
  agent_id       = "ocid1.genaiagent.oc1..aaaaaaaaexample"
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"

  guardrail_config {
    prompt_injection_config {
      input_guardrail_mode = "BLOCK"
    }
  }
}
