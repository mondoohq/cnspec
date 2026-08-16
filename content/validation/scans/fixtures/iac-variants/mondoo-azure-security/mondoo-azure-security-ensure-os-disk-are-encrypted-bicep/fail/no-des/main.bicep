// Non-compliant: encryptionAtHost covers the temp disk and host caches, but the
// OS disk names no diskEncryptionSet so it stays on the platform-managed key.
resource vm 'Microsoft.Compute/virtualMachines@2024-07-01' = {
  name: 'app-vm-01'
  location: 'eastus'
  properties: {
    hardwareProfile: {
      vmSize: 'Standard_D2s_v5'
    }
    securityProfile: {
      encryptionAtHost: true
    }
    storageProfile: {
      osDisk: {
        createOption: 'FromImage'
        managedDisk: {
          storageAccountType: 'Premium_LRS'
        }
      }
    }
  }
}
