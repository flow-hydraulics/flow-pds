import NonFungibleToken from 0x{{.NonFungibleToken}} 
import ExampleNFT from 0x{{.ExampleNFT}}
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

    pub resource SharedCapabilities {
        access(self) let withdrawCap: Capability<&{NonFungibleToken.Provider}>
        access(self) let operatorCap: Capability<&{IPackNFT.IOperator}>

        pub fun withdrawFromIssuer(withdrawID: UInt64): @NonFungibleToken.NFT {
            let c = self.withdrawCap.borrow() ?? panic("no such cap")
            return <- c.withdraw(withdrawID: withdrawID)
        }
        
        // TODO: maybe we do not need to specify the issuer here, should be the creator of the SharedCapabilities
        // this is also used in storing inside the NFT though
        pub fun mintPackNFT(commitHashes: [String], issuer: Address){
            var i = 0
            let c = self.operatorCap.borrow() ?? panic("no such cap")
            while i < commitHashes.length{
                c.mint(commitHash: commitHashes[i], issuer: issuer)
                i = i + 1
            }
        }
        
        pub fun revealPackNFT(packNFTId: UInt64, nftIds: [UInt64]) {
            let c = self.operatorCap.borrow() ?? panic("no such cap")
            c.reveal(id: packNFTId, nftIds: nftIds)
        }

        pub fun openPackNFT(packNFTId: UInt64, nftIds: [UInt64], owner: Address) {
            let c = self.operatorCap.borrow() ?? panic("no such cap")
            PDS.releaseEscrow(nftIds: nftIds, owner: owner)
            c.open(id: packNFTId)
        }
        

        init(
            withdrawCap: Capability<&{NonFungibleToken.Provider}>
            operatorCap: Capability<&{IPackNFT.IOperator}>

        ){
            self.withdrawCap = withdrawCap
            self.operatorCap = operatorCap
        }
    }

    pub var DistId: UInt64
    access(contract) let Distributions: @{UInt64: SharedCapabilities}

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
            let currentId = PDS.DistId
            PDS.Distributions[currentId] <-! sharedCap
            PDS.DistId = currentId + 1 
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
        
        pub fun mintPackNFT(distId: UInt64, commitHashes: [String], issuer: Address){
            assert(PDS.Distributions.containsKey(distId), message: "No such distribution")
            let d <- PDS.Distributions.remove(key: distId)!
            d.mintPackNFT(commitHashes: commitHashes, issuer: issuer)
            PDS.Distributions[distId] <-! d
        }
        
        pub fun revealPackNFT(distId: UInt64, packNFTId: UInt64, nftIds: [UInt64]){
            assert(PDS.Distributions.containsKey(distId), message: "No such distribution")
            let d <- PDS.Distributions.remove(key: distId)!
            d.revealPackNFT(packNFTId: packNFTId, nftIds: nftIds)
            PDS.Distributions[distId] <-! d
        }

    }
    
    access(contract) fun getManagerCollectionCap(escrowCollectionPublic: PublicPath): Capability<&{NonFungibleToken.CollectionPublic}> {
        let pdsCollection = self.account.getCapability<&{NonFungibleToken.CollectionPublic}>(escrowCollectionPublic)
        if !pdsCollection.check(){
            panic("Please ensure PDS has created and linked a Collection for recieving escrows")
        }
        return pdsCollection
    }
    
    access(contract) fun releaseEscrow(nftIds: [UInt64], owner: Address) {
        let pdsCollection = self.account.borrow<&ExampleNFT.Collection>(from: ExampleNFT.CollectionStoragePath) ?? panic("cannot find escrow collection")
        let recvAcct = getAccount(owner)
        let recv = recvAcct.getCapability(ExampleNFT.CollectionPublicPath).borrow<&{NonFungibleToken.CollectionPublic}>()
            ?? panic("Unable to borrow Collection Public reference for recipient")
        var i = 0
        while i < nftIds.length {
            recv.deposit(token: <- pdsCollection.withdraw(withdrawID: nftIds[i]))
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
        adminAccount: AuthAccount,
        packIssuerStoragePath: StoragePath,
        packIssuerCapRecv: PublicPath,
        distCreatorStoragePath: StoragePath,
        distCreatorPrivPath: PrivatePath,
        distManagerStoragePath: StoragePath,
        version: String
    ) {
        self.DistId = 0
        self.Distributions <- {}
        self.packIssuerStoragePath = packIssuerStoragePath
        self.packIssuerCapRecv = packIssuerCapRecv
        self.distCreatorStoragePath = distCreatorStoragePath
        self.distCreatorPrivPath = distCreatorPrivPath
        self.distManagerStoragePath = distManagerStoragePath
        self.version = version
        
        // Create a distributionCreator to share create capability with PackIssuer 
        let d <- create DistributionCreator()
        adminAccount.save(<-d, to: self.distCreatorStoragePath)
        adminAccount.link<&DistributionCreator{PDS.IDistCreator}>(self.distCreatorPrivPath, target: self.distCreatorStoragePath)

        // Create a distributionManager to manager distributions (withdraw for escrow, mint PackNFT todo: reveal / transfer) 
        let m <- create DistributionManager()
        adminAccount.save(<-m, to: self.distManagerStoragePath)
    }
}
