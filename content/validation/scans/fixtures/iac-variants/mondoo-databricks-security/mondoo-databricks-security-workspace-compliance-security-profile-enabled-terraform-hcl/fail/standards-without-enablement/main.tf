resource "databricks_compliance_security_profile_workspace_setting" "this" {
  compliance_security_profile_workspace {
    is_enabled           = false
    compliance_standards = ["HIPAA"]
  }
}
