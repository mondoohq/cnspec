# Non-compliant: output is moderated by neither setting.
resource "oci_generative_ai_agent_agent_endpoint" "example" {
  agent_id       = "ocid1.genaiagent.oc1..aaaaaaaaexample"
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"

  content_moderation_config {
    should_enable_on_input  = true
    should_enable_on_output = false
  }

  guardrail_config {
    content_moderation_config {
      input_guardrail_mode  = "BLOCK"
      output_guardrail_mode = "DISABLE"
    }
  }
}
