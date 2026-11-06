# Compliant: the older content moderation settings cover input and output.
resource "oci_generative_ai_agent_agent_endpoint" "example" {
  agent_id       = "ocid1.genaiagent.oc1..aaaaaaaaexample"
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"

  content_moderation_config {
    should_enable_on_input  = true
    should_enable_on_output = true
  }
}
