// This transactions deploys a contract with init args
//
transaction(
    contractName: String, 
    code: String,
    collectionStoragePath: StoragePath,
    collectionPublicPath: PublicPath,
    minterStoragePath: StoragePath,
    minterPrivPath: PrivatePath,
    minterProxyStoragePath: StoragePath,
    minterProxyMintCapRecv: PublicPath,
    version: String,
) {
    prepare(owner: AuthAccount) {
        let existingContract = owner.contracts.get(name: contractName)

        if (existingContract == nil) {
            log("no contract")
            owner.contracts.add(
                name: contractName, 
                code: code.decodeHex(), 
                owner,
                collectionStoragePath: collectionStoragePath,
                collectionPublicPath: collectionPublicPath,
                minterStoragePath: minterStoragePath,
                minterPrivPath: minterPrivPath,
                minterProxyStoragePath: minterProxyStoragePath,
                minterProxyMintCapRecv: minterProxyMintCapRecv,
                version: version,
            )
        } else {
            log("has contract")
            owner.contracts.update__experimental(name: contractName, code: code.decodeHex())
        }
    }
}
