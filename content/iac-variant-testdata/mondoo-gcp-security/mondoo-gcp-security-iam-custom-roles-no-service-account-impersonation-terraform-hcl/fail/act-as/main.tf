# actAs lets a holder of this role run workloads as any service account it can
# name, inheriting that account's privileges.
resource "google_project_iam_custom_role" "deployer" {
  project     = var.project_id
  role_id     = "pipelineDeployer"
  title       = "Pipeline Deployer"
  description = "Used by the deployment pipeline"
  stage       = "GA"

  permissions = [
    "compute.instances.create",
    "iam.serviceAccounts.actAs",
  ]
}
