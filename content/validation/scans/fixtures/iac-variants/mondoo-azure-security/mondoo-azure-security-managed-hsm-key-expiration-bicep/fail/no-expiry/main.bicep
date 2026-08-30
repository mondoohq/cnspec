// A key with no expiration stays valid indefinitely, so nothing forces a
// rotation or a review.
resource hsmKey 'Microsoft.KeyVault/managedHSMs/keys@2023-07-01' = {
  name: 'prod-hsm/payment-signing'
  properties: {
    kty: 'EC-HSM'
    curveName: 'P-256'
    attributes: {
      enabled: true
    }
  }
}
