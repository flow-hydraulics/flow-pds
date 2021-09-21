// import NonFungibleToken from "./NonFungibleToken.cdc" 
// import IPackNFT from "./IPackNFT.cdc" 
import NonFungibleToken from 0x{{.NonFungibleToken}} 
import IPackNFT from 0x{{.IPackNFT}} 

pub contract PDS{
    /// The collection to hold all escrowed NFT
    /// Original collection created from PackNFT
    pub var version: String
    pub let packCollectionPubPath: PublicPath
    pub let packIssuerStoragePath: StoragePath 
    pub let packIssuerCapRecv: PublicPath 
    pub let distCreatorStoragePath: StoragePath
    pub let distCreatorPrivPath: PrivatePath
    pub let distManagerStoragePath: StoragePath

    pub resource SharedCapabilities {
        access(self) let withdrawCap: Capability<&{NonFungibleToken.Provider}>
        access(self) let mintCap: Capability<&{IPackNFT.IMinter}>

        pub fun withdrawFromIssuer(withdrawID: UInt64): @NonFungibleToken.NFT {
            let c = self.withdrawCap.borrow() ?? panic("no such cap")
            return <- c.withdraw(withdrawID: withdrawID)
        }
        
        // TODO: maybe we do not need to specify the issuer here, should be the creator of the SharedCapabilities
        pub fun mintPackNFT(commitHashes: [String], issuer: Address){
            var i = 0
            let c = self.mintCap.borrow() ?? panic("no such cap")
            while i < commitHashes.len{
                c.mint(commitHash: commitHashes[i], issuer: issuer)
                i = i + 1
            }
        }

        init(
            withdrawCap: Capability<&{NonFungibleToken.Provider}>
            mintCap: Capability<&{IPackNFT.IMinter}>

        ){
            self.withdrawCap = withdrawCap
            self.mintCap = mintCap
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
        // TODO: set state on PackNFT
        pub fun withdraw(distId: UInt64, nftIDs: [UInt64]) {
            assert(PDS.Distributions.containsKey(distId), message: "No such distribution")
            let d <- PDS.Distributions.remove(key: distId)!
            let pdsCollection = PDS.getManagerCollectionCap().borrow()!
            var i = 0
            while i < nftID.length {
                let nft <- d.withdrawFromIssuer(withdrawID: nftID[i])
                pdsCollection.deposit(token:<-nft)
            } 
            PDS.Distributions[distId] <-! d
        }
        
        // TODO: set state on PackNFT, maybe remove issuer (need backend discussion)
        pub fun mintPackNFT(distId: UInt64, commitHashes: [String], issuer: Address){
            assert(PDS.Distributions.containsKey(distId), message: "No such distribution")
            let d <- PDS.Distributions.remove(key: distId)!
            d.mintPackNFT(commitHashes: commitHashes, issuer: issuer)
            PDS.Distributions[distId] <-! d
        }

    }
    
    access(contract) fun getManagerCollectionCap(): Capability<&{NonFungibleToken.CollectionPublic}> {
        let pdsCollection = self.account.getCapability<&{NonFungibleToken.CollectionPublic}>(self.packCollectionPubPath)
        if !pdsCollection.check(){
            panic("Please ensure you create and link a Collection for recieving escrows")
        }
        return pdsCollection
    }
    
    pub fun createPackIssuer (): @PackIssuer{
        return <- create PackIssuer()
    }

    pub fun createSharedCapabilities (
            withdrawCap: Capability<&{NonFungibleToken.Provider}>
            mintCap: Capability<&{IPackNFT.IMinter}>
    ): @SharedCapabilities{
        return <- create SharedCapabilities(
            withdrawCap: withdrawCap,
            mintCap: mintCap
        )
    }
    
    init(
        adminAccount: AuthAccount,
        packCollectionPubPath: PublicPath,
        packIssuerStoragePath: StoragePath,
        packIssuerCapRecv: PublicPath,
        distCreatorStoragePath: StoragePath,
        distCreatorPrivPath: PrivatePath,
        distManagerStoragePath: StoragePath,
        version: String
    ) {
        self.DistId = 0
        self.Distributions <- {}
        self.packCollectionPubPath = packCollectionPubPath
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
