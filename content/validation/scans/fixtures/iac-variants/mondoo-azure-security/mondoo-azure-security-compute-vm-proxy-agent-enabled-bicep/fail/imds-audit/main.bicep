// The Instance Metadata Service is only audited, so any process on the VM
// still gets a managed identity token.
resource vm 'Microsoft.Compute/virtualMachines@2024-11-01' = {
  name: 'app-vm'
  location: 'eastus'
  properties: {
    hardwareProfile: {
      vmSize: 'Standard_D2s_v5'
    }
    securityProfile: {
      securityType: 'TrustedLaunch'
      proxyAgentSettings: {
        enabled: true
        wireServer: {
          mode: 'Enforce'
        }
        imds: {
          mode: 'Audit'
        }
      }
    }
  }
}
