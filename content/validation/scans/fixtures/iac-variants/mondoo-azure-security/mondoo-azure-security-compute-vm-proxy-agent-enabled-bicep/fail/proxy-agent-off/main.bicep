// Without the metadata proxy agent, any process on the VM can reach IMDS and
// read the managed identity token directly.
resource vm 'Microsoft.Compute/virtualMachines@2024-03-01' = {
  name: 'app-vm'
  location: 'eastus'
  properties: {
    hardwareProfile: {
      vmSize: 'Standard_D2s_v5'
    }
    securityProfile: {
      securityType: 'TrustedLaunch'
      proxyAgentSettings: {
        enabled: false
      }
    }
  }
}
