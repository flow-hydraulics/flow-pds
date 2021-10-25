// This transactions deploys a contract with init args
//
transaction(
    contractName: String, 
    code: String,
    CollectionStoragePath: StoragePath,
    CollectionPublicPath: PublicPath,
    CollectionIPackNFTPublicPath: PublicPath,
    OperatorStoragePath: StoragePath,
    OperatorPrivPath: PrivatePath,
    version: String,
) {
    prepare(owner: AuthAccount) {
        let existingContract = owner.contracts.get(name: contractName)

        if (existingContract == nil) {
            log("no contract")
            owner.contracts.add(
                name: contractName, 
                code: code.decodeHex(), 
                CollectionStoragePath: CollectionStoragePath,
                CollectionPublicPath: CollectionPublicPath,
                CollectionIPackNFTPublicPath: CollectionIPackNFTPublicPath,
                OperatorStoragePath: OperatorStoragePath,
                OperatorPrivPath: OperatorPrivPath,
                version: version,
            )
        } else {
            log("has contract")
            owner.contracts.update__experimental(name: contractName, code: code.decodeHex())
        }
    }
}
