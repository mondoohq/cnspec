resource association 'Microsoft.Network/networkSecurityPerimeters/resourceAssociations@2023-08-01-preview' = {
  name: 'storage-association'
  properties: {
    accessMode: 'Enforced'
    privateLinkResource: {
      id: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/prod/providers/Microsoft.Storage/storageAccounts/exampledata'
    }
  }
}
