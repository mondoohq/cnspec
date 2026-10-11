// Current API versions set the mode per endpoint; both enforce.
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
          mode: 'Enforce'
        }
      }
    }
  }
}
