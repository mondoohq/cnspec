// Learning mode observes and logs what the perimeter would block without
// blocking anything, so the resource stays as reachable as before.
resource association 'Microsoft.Network/networkSecurityPerimeters/resourceAssociations@2023-08-01-preview' = {
  name: 'prod-perimeter/storage-association'
  properties: {
    accessMode: 'Learning'
    privateLinkResource: {
      id: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/prod/providers/Microsoft.Storage/storageAccounts/exampledata'
    }
  }
}
