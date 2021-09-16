// This transactions deploys a contract with init args
//
transaction(
    contractName: String, 
    code: String,
    collectionStoragePath: StoragePath,
    collectionPublicPath: PublicPath,
    minterStoragePath: StoragePath,
    minterPrivPath: PrivatePath,
    version: String,
) {
    prepare(owner: AuthAccount) {
        let existingContract = owner.contracts.get(name: contractName)

        if (existingContract == nil) {
            owner.contracts.add(
                name: contractName, 
                code: code.decodeHex(), 
                owner,
                collectionStoragePath: collectionStoragePath,
                collectionPublicPath: collectionPublicPath,
                minterStoragePath: minterStoragePath,
                minterPrivPath: minterPrivPath,
                version: version,
            )
        } else {
            owner.contracts.update__experimental(name: contractName, code: code.decodeHex())
        }
    }
}
h
