// This transactions deploys a contract with init args
//
transaction(
    contractName: String, 
    code: String,
    PackIssuerStoragePath: StoragePath,
    PackIssuerCapRecv: PublicPath,
    DistCreatorStoragePath: StoragePath,
    DistCreatorPrivPath: PrivatePath,
    DistManagerStoragePath: StoragePath,
    version: String,
) {
    prepare(owner: AuthAccount) {
        let existingContract = owner.contracts.get(name: contractName)

        if (existingContract == nil) {
            log("no contract")
            owner.contracts.add(
                name: contractName, 
                code: code.decodeHex(), 
                PackIssuerStoragePath: PackIssuerStoragePath,
                PackIssuerCapRecv: PackIssuerCapRecv,
                DistCreatorStoragePath: DistCreatorStoragePath,
                DistCreatorPrivPath: DistCreatorPrivPath,
                DistManagerStoragePath: DistManagerStoragePath,
                version: version,
            )
        } else {
            log("has contract")
            owner.contracts.update__experimental(name: contractName, code: code.decodeHex())
        }
    }
}
