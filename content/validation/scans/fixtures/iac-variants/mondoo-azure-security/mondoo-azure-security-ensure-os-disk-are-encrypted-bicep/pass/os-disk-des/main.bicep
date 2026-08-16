resource vm 'Microsoft.Compute/virtualMachines@2024-07-01' = {
  name: 'app-vm-01'
  location: 'eastus'
  properties: {
    hardwareProfile: {
      vmSize: 'Standard_D2s_v5'
    }
    storageProfile: {
      osDisk: {
        createOption: 'FromImage'
        managedDisk: {
          storageAccountType: 'Premium_LRS'
          diskEncryptionSet: {
            id: '/subscriptions/x/resourceGroups/rg/providers/Microsoft.Compute/diskEncryptionSets/des1'
          }
        }
      }
    }
  }
}
