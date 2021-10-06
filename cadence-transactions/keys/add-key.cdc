// Transaction for adding a public key exported from Google KMS to an account (signer)

transaction(publicKey: String) {
    prepare(signer: AuthAccount) {
        let key = PublicKey(
            publicKey: publicKey.decodeHex(),
            signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
        )

        // NOTE: Using SHA2_256 for Google KMS keys
        signer.keys.add(
            publicKey: key,
            hashAlgorithm: HashAlgorithm.SHA2_256,
            weight: 1000.0
        )
    }
}
