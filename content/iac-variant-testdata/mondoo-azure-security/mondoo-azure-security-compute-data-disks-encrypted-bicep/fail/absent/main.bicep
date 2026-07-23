resource dataDisk 'Microsoft.Compute/disks@2023-04-02' = {
  name: 'app-data-disk-01'
  location: 'eastus'
  sku: {
    name: 'Premium_LRS'
  }
  properties: {
    diskSizeGB: 256
    creationData: {
      createOption: 'Empty'
    }
  }
}
