import NonFungibleToken from 0x{{.NonFungibleToken}} 
import IPackNFT from 0x{{.IPackNFT}} 

pub contract PDS{
    /// The collection to hold all escrowed NFT
    /// Original collection created from PackNFT
    pub var version: String
    pub let packIssuerStoragePath: StoragePath 
    pub let packIssuerCapRecv: PublicPath 
    pub let distCreatorStoragePath: StoragePath
    pub let distCreatorPrivPath: PrivatePath
    pub let distManagerStoragePath: StoragePath

    pub var nextDistId: UInt64
    access(contract) let Distributions: @{UInt64: SharedCapabilities}

    pub struct Collectible: IPackNFT.Collectible {
        pub let address: Address
        pub let contractName: String
        pub let id: UInt64

        // returning in string so that it is more readable and anyone can check the hash
        pub fun hashString(): String {
            // address string is 16 characters long with 0x as prefix (for 8 bytes in hex)
            // example: ,f3fcd2c1a78f5ee.ExampleNFT.12
            let c = "A."
            var a = ""
            let addrStr = self.address.toString()
            if addrStr.length < 18 {
                let padding = 18 - addrStr.length
                let p = "0"
                var i = 0
                a = addrStr.slice(from: 2, upTo: addrStr.length)
                while i < padding {
                    a = p.concat(a)
                    i = i + 1
                }
            } else {
                a = addrStr.slice(from: 2, upTo: 18)
            }
            var str = c.concat(a).concat(".").concat(self.contractName).concat(".").concat(self.id.toString())
            return str
        } 
        init(address: Address, contractName: String, id: UInt64) {
            self.address = address
            self.contractName = contractName
            self.id = id
        }
    }

    pub resource SharedCapabilities {
        access(self) let withdrawCap: Capability<&{NonFungibleToken.Provider}>
        access(self) let operatorCap: Capability<&{IPackNFT.IOperator}>

        pub fun withdrawFromIssuer(withdrawID: UInt64): @NonFungibleToken.NFT {
            let c = self.withdrawCap.borrow() ?? panic("no such cap")
            return <- c.withdraw(withdrawID: withdrawID)
        }
        
        pub fun mintPackNFT(distId: UInt64, commitHashes: [String], issuer: Address, recvCap: &{NonFungibleToken.CollectionPublic} ){
            var i = 0
            let c = self.operatorCap.borrow() ?? panic("no such cap")
            while i < commitHashes.length{
                let nft <- c.mint(distId: distId, commitHash: commitHashes[i], issuer: issuer)
                i = i + 1
                let n <- nft as! @NonFungibleToken.NFT
                recvCap.deposit(token: <- n)
            }
        }
        
        pub fun revealPackNFT(packId: UInt64, nfts: [{IPackNFT.Collectible}], salt: String) {
            let c = self.operatorCap.borrow() ?? panic("no such cap")
            c.reveal(id: packId, nfts: nfts, salt: salt)
        }

        pub fun openPackNFT(packId: UInt64, nftIds: [UInt64], recvCap: &{NonFungibleToken.CollectionPublic}, collectionProviderPath: PrivatePath) {
            let c = self.operatorCap.borrow() ?? panic("no such cap")
            // This checks and sets the status of the pack before releasing escrow 
            c.open(id: packId)
            PDS.releaseEscrow(nftIds: nftIds, recvCap: recvCap , collectionProviderPath: collectionProviderPath)
        }
        

        init(
            withdrawCap: Capability<&{NonFungibleToken.Provider}>
            operatorCap: Capability<&{IPackNFT.IOperator}>

        ){
            self.withdrawCap = withdrawCap
            self.operatorCap = operatorCap
        }
    }


    /// Issuer has created a distribution 
    pub event DistributionCreated(DistId: UInt64)
    
    pub resource interface PackIssuerCapReciever {
        pub fun setDistCap(cap: Capability<&DistributionCreator{IDistCreator}>) 
    }
    
    pub resource PackIssuer: PackIssuerCapReciever {
        access(self) var cap:  Capability<&DistributionCreator{IDistCreator}>?
        
        pub fun setDistCap(cap: Capability<&DistributionCreator{IDistCreator}>) {
            pre {
                cap.borrow() != nil: "Invalid capability"
            }
            self.cap = cap 
        }

        pub fun create(sharedCap: @SharedCapabilities) {
            let c = self.cap!.borrow()!
            c.createNewDist(sharedCap: <- sharedCap)
        }
        init() {
            self.cap = nil
        }
    }

    // DistCap to be shared
    pub resource interface  IDistCreator {
        pub fun createNewDist(sharedCap: @SharedCapabilities) 
    }

    pub resource DistributionCreator: IDistCreator {
        pub fun createNewDist(sharedCap: @SharedCapabilities) {
            let currentId = PDS.nextDistId
            PDS.Distributions[currentId] <-! sharedCap
            PDS.nextDistId = currentId + 1 
            emit DistributionCreated(DistId: currentId)
        }
    }
    
    pub resource DistributionManager {
        pub fun withdraw(distId: UInt64, nftIDs: [UInt64], escrowCollectionPublic: PublicPath) {
            assert(PDS.Distributions.containsKey(distId), message: "No such distribution")
            let d <- PDS.Distributions.remove(key: distId)!
            let pdsCollection = PDS.getManagerCollectionCap(escrowCollectionPublic: escrowCollectionPublic).borrow()!
            var i = 0
            while i < nftIDs.length {
                let nft <- d.withdrawFromIssuer(withdrawID: nftIDs[i])
                pdsCollection.deposit(token:<-nft)
                i = i + 1
            } 
            PDS.Distributions[distId] <-! d
        }
        
        pub fun mintPackNFT(distId: UInt64, commitHashes: [String], issuer: Address, recvCap: &{NonFungibleToken.CollectionPublic}){
            assert(PDS.Distributions.containsKey(distId), message: "No such distribution")
            let d <- PDS.Distributions.remove(key: distId)!
            d.mintPackNFT(distId: distId, commitHashes: commitHashes, issuer: issuer, recvCap: recvCap)
            PDS.Distributions[distId] <-! d
        }
        
        pub fun revealPackNFT(distId: UInt64, packId: UInt64, nftContractAddrs: [Address], nftContractName: [String], nftIds: [UInt64], salt: String){
            assert(PDS.Distributions.containsKey(distId), message: "No such distribution")
            assert(
                nftContractAddrs.length == nftContractName.length && 
                nftContractName.length == nftIds.length, 
                message: "NFTs must be fully described"
            )
            let d <- PDS.Distributions.remove(key: distId)!
            let arr: [{IPackNFT.Collectible}] = []
            var i = 0
            while i < nftContractAddrs.length {
                let s = Collectible(address: nftContractAddrs[i], contractName: nftContractName[i], id: nftIds[i])
                arr.append(s)
                i = i + 1
            }
            d.revealPackNFT(packId: packId, nfts: arr, salt: salt)
            PDS.Distributions[distId] <-! d
        }

        pub fun openPackNFT(distId: UInt64, packId: UInt64, nftIds: [UInt64], recvCap: &{NonFungibleToken.CollectionPublic}, collectionProviderPath: PrivatePath){
            assert(PDS.Distributions.containsKey(distId), message: "No such distribution")
            let d <- PDS.Distributions.remove(key: distId)!
            d.openPackNFT(packId: packId, nftIds: nftIds, recvCap: recvCap, collectionProviderPath: collectionProviderPath)
            PDS.Distributions[distId] <-! d
        }

    }
    
    access(contract) fun getManagerCollectionCap(escrowCollectionPublic: PublicPath): Capability<&{NonFungibleToken.CollectionPublic}> {
        let pdsCollection = self.account.getCapability<&{NonFungibleToken.CollectionPublic}>(escrowCollectionPublic)
        assert(pdsCollection.check(), message: "Please ensure PDS has created and linked a Collection for recieving escrows")
        return pdsCollection
    }
    
    access(contract) fun releaseEscrow(nftIds: [UInt64], recvCap:  &{NonFungibleToken.CollectionPublic}, collectionProviderPath: PrivatePath ) {
        let pdsCollection = self.account.getCapability(collectionProviderPath).borrow<&{NonFungibleToken.Provider}>()
            ?? panic("Unable to borrow PDS collection provider capability from private path")
        var i = 0
        while i < nftIds.length {
            recvCap.deposit(token: <- pdsCollection.withdraw(withdrawID: nftIds[i]))
            i = i + 1
        }
    }

    pub fun createPackIssuer (): @PackIssuer{
        return <- create PackIssuer()
    }

    pub fun createSharedCapabilities (
            withdrawCap: Capability<&{NonFungibleToken.Provider}>
            operatorCap: Capability<&{IPackNFT.IOperator}>
    ): @SharedCapabilities{
        return <- create SharedCapabilities(
            withdrawCap: withdrawCap,
            operatorCap: operatorCap
        )
    }

    
    init(
        packIssuerStoragePath: StoragePath,
        packIssuerCapRecv: PublicPath,
        distCreatorStoragePath: StoragePath,
        distCreatorPrivPath: PrivatePath,
        distManagerStoragePath: StoragePath,
        version: String
    ) {
        self.nextDistId = 0
        self.Distributions <- {}
        self.packIssuerStoragePath = packIssuerStoragePath
        self.packIssuerCapRecv = packIssuerCapRecv
        self.distCreatorStoragePath = distCreatorStoragePath
        self.distCreatorPrivPath = distCreatorPrivPath
        self.distManagerStoragePath = distManagerStoragePath
        self.version = version
        
        // Create a distributionCreator to share create capability with PackIssuer 
        let d <- create DistributionCreator()
        self.account.save(<-d, to: self.distCreatorStoragePath)
        self.account.link<&DistributionCreator{PDS.IDistCreator}>(self.distCreatorPrivPath, target: self.distCreatorStoragePath)

        // Create a distributionManager to manager distributions (withdraw for escrow, mint PackNFT todo: reveal / transfer) 
        let m <- create DistributionManager()
        self.account.save(<-m, to: self.distManagerStoragePath)
    }
}
