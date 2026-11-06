# Compliant: input is moderated by the older setting and output by the guardrail.
resource "oci_generative_ai_agent_agent_endpoint" "example" {
  agent_id       = "ocid1.genaiagent.oc1..aaaaaaaaexample"
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"

  content_moderation_config {
    should_enable_on_input = true
  }

  guardrail_config {
    content_moderation_config {
      output_guardrail_mode = "BLOCK"
    }
  }
}
