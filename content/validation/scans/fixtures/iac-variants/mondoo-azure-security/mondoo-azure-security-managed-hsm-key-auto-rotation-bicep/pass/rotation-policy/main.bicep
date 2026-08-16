resource hsmKey 'Microsoft.KeyVault/managedHSMs/keys@2023-07-01' = {
  name: 'payment-signing'
  properties: {
    kty: 'EC-HSM'
    curveName: 'P-256'
    keyOps: [
      'sign'
      'verify'
    ]
    rotationPolicy: {
      lifetimeActions: [
        {
          trigger: {
            timeBeforeExpiry: 'P30D'
          }
          action: {
            type: 'Rotate'
          }
        }
      ]
    }
  }
}
