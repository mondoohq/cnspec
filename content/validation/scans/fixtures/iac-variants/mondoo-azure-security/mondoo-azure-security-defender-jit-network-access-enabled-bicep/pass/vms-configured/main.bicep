resource jitPolicy 'Microsoft.Security/locations/jitNetworkAccessPolicies@2020-01-01' = {
  name: 'default'
  properties: {
    virtualMachines: [
      {
        id: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/prod/providers/Microsoft.Compute/virtualMachines/app-vm'
        ports: [
          {
            number: 22
            protocol: 'TCP'
            allowedSourceAddressPrefix: '203.0.113.0/24'
            maxRequestAccessDuration: 'PT3H'
          }
        ]
      }
    ]
  }
}
