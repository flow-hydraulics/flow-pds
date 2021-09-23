import Crypto
import NonFungibleToken from 0x{{.NonFungibleToken}} 
import IPackNFT from 0x{{.IPackNFT}} 

pub contract PackNFT: NonFungibleToken, IPackNFT {
    
    pub var totalSupply: UInt64
    pub let version: String 
    pub let collectionStoragePath: StoragePath 
    pub let collectionPublicPath: PublicPath 
    pub let collectionIPackNFTPublicPath: PublicPath
    pub let operatorStoragePath: StoragePath 
    pub let operatorPrivPath: PrivatePath 

    access(contract) let status: {UInt64: String}
    access(contract) let content: {UInt64: [UInt64]}

    pub event RevealRequest(id: UInt64)
    pub event OpenRequest(id: UInt64) 
    pub event Revealed(id: UInt64, nftIds: [UInt64])
    pub event Opened(id: UInt64)
    pub event Mint(id: UInt64, commitHash: String) 
    pub event ContractInitialized()
    pub event Withdraw(id: UInt64, from: Address?)
    pub event Deposit(id: UInt64, to: Address?)

    pub resource PackNFTOperator: IPackNFT.IOperator {
         pub fun mint(commitHash: String, issuer: Address) {
            let id = PackNFT.totalSupply
            let pack <- create NFT(initID: id, commitHash: commitHash, issuer: issuer)
            PackNFT.totalSupply = id + 1
            let recvAcct = getAccount(issuer)
            let recv = recvAcct.getCapability(PackNFT.collectionPublicPath).borrow<&{NonFungibleToken.CollectionPublic}>()
                ?? panic("Unable to borrow Collection Public reference for recipient")

            recv.deposit(token: <- pack)
            PackNFT.status.insert(key: id, "Sealed")
            emit Mint(id: id, commitHash: commitHash)
         }

        pub fun reveal(id: UInt64, nftIds: [UInt64]) {
            PackNFT.reveal(id: id, nftIds: nftIds)
        }

        pub fun open(id: UInt64) {
            PackNFT.open(id: id)
        }

         init(){}
    }
    
    pub resource NFT: NonFungibleToken.INFT, IPackNFT.IPackNFTToken, IPackNFT.IPackNFTOwnerOperator {
        pub let id: UInt64
        pub let commitHash: String
        pub let issuer: Address

        pub fun reveal(){
            PackNFT.revealRequest(id: self.id)
        }
        
        pub fun open(){
            PackNFT.openRequest(id: self.id)
        }

        init(initID: UInt64, commitHash: String, issuer: Address ) {
            self.id = initID
            self.commitHash = commitHash
            self.issuer = issuer
        }

    }
    
       pub resource Collection: NonFungibleToken.Provider, NonFungibleToken.Receiver, NonFungibleToken.CollectionPublic, IPackNFT.IPackNFTCollection {
        // dictionary of NFT conforming tokens
        // NFT is a resource type with an `UInt64` ID field
        pub var ownedNFTs: @{UInt64: NonFungibleToken.NFT}

        init () {
            self.ownedNFTs <- {}
        }

        // withdraw removes an NFT from the collection and moves it to the caller
        pub fun withdraw(withdrawID: UInt64): @NonFungibleToken.NFT {
            let token <- self.ownedNFTs.remove(key: withdrawID) ?? panic("missing NFT")
            emit Withdraw(id: token.id, from: self.owner?.address)
            let nonfungibleToken <- token

            return <- nonfungibleToken
        }

        // deposit takes a NFT and adds it to the collections dictionary
        // and adds the ID to the id array
        pub fun deposit(token: @NonFungibleToken.NFT) {
            let token <- token as! @PackNFT.NFT

            let id: UInt64 = token.id

            // add the new token to the dictionary which removes the old one
            let oldToken <- self.ownedNFTs[id] <- token 
            emit Deposit(id: id, to: self.owner?.address)

            destroy oldToken
        }

        // getIDs returns an array of the IDs that are in the collection
        pub fun getIDs(): [UInt64] {
            return self.ownedNFTs.keys
        }

        // borrowNFT gets a reference to an NFT in the collection
        // so that the caller can read its metadata and call its methods
        pub fun borrowNFT(id: UInt64): &NonFungibleToken.NFT {
            return &self.ownedNFTs[id] as &NonFungibleToken.NFT
        }
        
        pub fun borrowPackNFT(id: UInt64): &IPackNFT.NFT {
            let nft<- self.ownedNFTs.remove(key: id) ?? panic("missing NFT")
            let token <- nft as! @PackNFT.NFT
            let ref = &token as &IPackNFT.NFT
            self.ownedNFTs[id] <-! token as! @PackNFT.NFT
            return ref 
        }

        destroy() {
            destroy self.ownedNFTs
        }
    }
    
    access(contract) fun revealRequest(id: UInt64 ) {
        emit RevealRequest(id: id)
    }

    access(contract) fun openRequest(id: UInt64) {
        emit OpenRequest(id: id)
    }

    access(contract) fun reveal(id: UInt64, nftIds: [UInt64]) {
        self.status[id] = "Revealed"
        self.content.insert(key: id, nftIds)
        emit Revealed(id: id, nftIds: nftIds)
    }

    access(contract) fun open(id: UInt64) {
        self.status[id] = "Opened"
        emit Opened(id: id)
    }
    
    
    pub fun getStatus(id: UInt64): String? {
        return self.status[id]
    }
    
    pub fun createEmptyCollection(): @NonFungibleToken.Collection {
        let c <- create Collection()
        return <- c
    }  
    
    init(
        adminAccount: AuthAccount, 
        collectionStoragePath: StoragePath,
        collectionPublicPath: PublicPath,
        collectionIPackNFTPublicPath: PublicPath,
        operatorStoragePath: StoragePath,
        operatorPrivPath: PrivatePath,
        version: String
    ){
        self.totalSupply = 0
        self.status = {}
        self.content = {}
        self.collectionStoragePath = collectionStoragePath
        self.collectionPublicPath = collectionPublicPath
        self.collectionIPackNFTPublicPath = collectionIPackNFTPublicPath
        self.operatorStoragePath = operatorStoragePath
        self.operatorPrivPath = operatorPrivPath
        self.version = version

        // Create a collection to receive Pack NFTs
        let collection <- create Collection()
        adminAccount.save(<-collection, to: self.collectionStoragePath)
        adminAccount.link<&Collection{NonFungibleToken.CollectionPublic}>(self.collectionPublicPath, target: self.collectionStoragePath)
        adminAccount.link<&Collection{IPackNFT.IPackNFTCollection}>(self.collectionIPackNFTPublicPath, target: self.collectionStoragePath)

        // Create a operator to share mint capability with proxy 
        let operator <- create PackNFTOperator()
        adminAccount.save(<-operator, to: self.operatorStoragePath)
        adminAccount.link<&PackNFTOperator{IPackNFT.IOperator}>(self.operatorPrivPath, target: self.operatorStoragePath)
    }

}
 
