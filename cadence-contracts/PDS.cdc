import NonFungibleToken from "./NonFungibleToken.cdc"
import IPackNFT from "./IPackNFT.cdc"

pub contract PDS{
    /// The collection to hold all escrowed NFT
    /// Original collection created from PackNFT
    pub let packCollectionPubPath: PublicPath

    pub resource SharedCapabilities {
        pub let account: Address
        access(self) let withdrawCap: Capability<&AnyResource{NonFungibleToken.Provider}>

        init(
            account: Address
            withdrawCap: Capability<&AnyResource{NonFungibleToken.Provider}>

        ){
            self.account = account
            self.withdrawCap = withdrawCap
        }
    }
    pub var DistId: UInt64
    access(contract) let Distributions: @{UInt64: SharedCapabilities}

    /// Issuer has created a distribution 
    pub event DistributionCreated(DistId: UInt64)
    
    pub resource interface PackIssuerCapReciever {
        
    }
    
    pub resource PackIssuer {
        access(self) var cap:  Capability<&DistributionCreator>?
        
        pub fun setDistCap(sharedCap: Capability<&DistributionCreator>) {
            pre {
                sharedCap.borrow() != nil: "Invalid capability"
            }
            self.cap = sharedCap
        }

        pub fun create(sharedCap: @SharedCapabilities) {
            let c = self.cap!.borrow()!
            c.createNewDist(sharedCap: <- sharedCap)
        }
        
        init() {
            self.cap <- nil
        }
 
    }

    // DistCap
    pub resource DistributionCreator {
        pub fun createNewDist(sharedCap: @SharedCapabilities) {
            let currentId = DistId
            PDS.Distributions[currentId] <-! shareCapbilities 
            DistId = currentId + 1 
            emit DistId(DistId: DistId)
        }
    }

    pub fun withdraw(id: UInt64, nftID: [UInt64]) {
        assert(self.Distributions.containsKey(id), message: "No such distribution")
        let d <- self.Distributions.remove(key: id)!
        let pdsCollection = self.account.getCapability(self.packCollectionPubPath)!
            .borrow(&{NonFungibleToken.CollectionPublic})
            ?? panic ("pds accoutn does not have a collection")
        var i = 0
        while i < nftID.length {
            let nft <- d.withdraw(withdrawID: nftID[i])
            pdsCollection.deposit(<-nft)
        } 
        self.distribution[id] <-! d
    }
    
    
    init(packCollectionPubPath: PublicPath) {
        self.DistId = 0
        self.Distributions <- {}
        self.packCollectionPubPath = packCollectionPubPath
    }
}
