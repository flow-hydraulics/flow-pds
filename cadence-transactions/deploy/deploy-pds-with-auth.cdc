// This transactions deploys a contract with init args
//
transaction(
    contractName: String, 
    code: String,
    packIssuerStoragePath: StoragePath,
    packIssuerCapRecv: PublicPath,
    distCreatorStoragePath: StoragePath,
    distCreatorPrivPath: PrivatePath,
    distManagerStoragePath: StoragePath,
    version: String,
) {
    prepare(owner: AuthAccount) {
        let existingContract = owner.contracts.get(name: contractName)

        if (existingContract == nil) {
            log("no contract")
            owner.contracts.add(
                name: contractName, 
                code: code.decodeHex(), 
                packIssuerStoragePath: packIssuerStoragePath,
                packIssuerCapRecv: packIssuerCapRecv,
                distCreatorStoragePath: distCreatorStoragePath,
                distCreatorPrivPath: distCreatorPrivPath,
                distManagerStoragePath: distManagerStoragePath,
                version: version,
            )
        } else {
            log("has contract")
            owner.contracts.update__experimental(name: contractName, code: code.decodeHex())
        }
    }
}
