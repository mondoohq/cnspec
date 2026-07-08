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
    encryption: {
      type: 'EncryptionAtRestWithCustomerKey'
      diskEncryptionSetId: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-security/providers/Microsoft.Compute/diskEncryptionSets/des-cmk'
    }
  }
}
