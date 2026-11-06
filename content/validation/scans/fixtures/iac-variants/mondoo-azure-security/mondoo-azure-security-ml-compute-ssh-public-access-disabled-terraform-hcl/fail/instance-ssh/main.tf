resource "azurerm_machine_learning_compute_instance" "example" {
  name                          = "example-instance"
  machine_learning_workspace_id = "/subscriptions/000/resourceGroups/rg/providers/Microsoft.MachineLearningServices/workspaces/ws"
  virtual_machine_size          = "STANDARD_DS2_V2"

  ssh {
    public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC example"
  }
}
