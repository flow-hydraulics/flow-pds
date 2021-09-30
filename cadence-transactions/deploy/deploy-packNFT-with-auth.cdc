// This transactions deploys a contract with init args
//
transaction(
    contractName: String, 
    code: String,
    collectionStoragePath: StoragePath,
    collectionPublicPath: PublicPath,
    collectionIPackNFTPublicPath: PublicPath,
    operatorStoragePath: StoragePath,
    operatorPrivPath: PrivatePath,
    version: String,
) {
    prepare(owner: AuthAccount) {
        let existingContract = owner.contracts.get(name: contractName)

        if (existingContract == nil) {
            log("no contract")
            owner.contracts.add(
                name: contractName, 
                code: code.decodeHex(), 
                collectionStoragePath: collectionStoragePath,
                collectionPublicPath: collectionPublicPath,
                collectionIPackNFTPublicPath: collectionIPackNFTPublicPath,
                operatorStoragePath: operatorStoragePath,
                operatorPrivPath: operatorPrivPath,
                version: version,
            )
        } else {
            log("has contract")
            owner.contracts.update__experimental(name: contractName, code: code.decodeHex())
        }
    }
}
