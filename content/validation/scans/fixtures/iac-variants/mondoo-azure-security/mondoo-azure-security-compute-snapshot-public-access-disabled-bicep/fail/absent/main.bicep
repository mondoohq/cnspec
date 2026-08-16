resource snapshot 'Microsoft.Compute/snapshots@2023-04-02' = {
  name: 'app-data-snapshot-01'
  location: 'eastus'
  sku: {
    name: 'Standard_LRS'
  }
  properties: {
    creationData: {
      createOption: 'Copy'
      sourceResourceId: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-compute/providers/Microsoft.Compute/disks/app-data-disk-01'
    }
  }
}
