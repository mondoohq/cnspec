resource vm 'Microsoft.Compute/virtualMachines@2023-09-01' = {
  name: 'linux-vm-01'
  location: 'eastus'
  properties: {
    hardwareProfile: {
      vmSize: 'Standard_D2s_v5'
    }
    storageProfile: {
      imageReference: {
        publisher: 'Canonical'
        offer: '0001-com-ubuntu-server-jammy'
        sku: '22_04-lts-gen2'
        version: 'latest'
      }
      osDisk: {
        createOption: 'FromImage'
        managedDisk: {
          storageAccountType: 'Premium_LRS'
        }
      }
    }
    osProfile: {
      computerName: 'linux-vm-01'
      adminUsername: 'azureuser'
      adminPassword: 'P@ssw0rd1234!'
      linuxConfiguration: {
        disablePasswordAuthentication: false
      }
    }
  }
}
