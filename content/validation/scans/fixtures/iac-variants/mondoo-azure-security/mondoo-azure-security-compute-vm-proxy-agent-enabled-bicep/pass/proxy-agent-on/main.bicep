resource vm 'Microsoft.Compute/virtualMachines@2024-03-01' = {
  name: 'app-vm'
  location: 'eastus'
  properties: {
    hardwareProfile: {
      vmSize: 'Standard_D2s_v5'
    }
    securityProfile: {
      securityType: 'TrustedLaunch'
      encryptionAtHost: true
      proxyAgentSettings: {
        enabled: true
        mode: 'Enforce'
      }
    }
  }
}
