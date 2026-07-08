resource vm 'Microsoft.Compute/virtualMachines@2023-09-01' = {
  name: 'app-vm-01'
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
        name: 'app-vm-01-osdisk'
        createOption: 'FromImage'
        caching: 'ReadWrite'
        vhd: {
          uri: 'https://contosostorage.blob.core.windows.net/vhds/app-vm-01-osdisk.vhd'
        }
      }
    }
    osProfile: {
      computerName: 'app-vm-01'
      adminUsername: 'azureuser'
    }
  }
}
