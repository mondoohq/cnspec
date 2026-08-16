resource osDisk 'Microsoft.Compute/disks@2023-04-02' = {
  name: 'vm-os-disk-01'
  location: 'eastus'
  sku: {
    name: 'Premium_LRS'
  }
  properties: {
    diskSizeGB: 128
    creationData: {
      createOption: 'Empty'
    }
    publicNetworkAccess: 'Enabled'
    networkAccessPolicy: 'AllowAll'
  }
}
