import Crypto
import NonFungibleToken from 0x{{.NonFungibleToken}} 
import IPackNFT from 0x{{.IPackNFT}} 

pub contract PackNFT: NonFungibleToken, IPackNFT {
    
    pub var totalSupply: UInt64
    pub let version: String 
    pub let collectionStoragePath: StoragePath 
    pub let collectionPublicPath: PublicPath 
    pub let minterStoragePath: StoragePath 
    pub let minterPrivPath: PrivatePath 
    pub let minterProxyStoragePath: StoragePath 
    pub let minterProxyMintCapRecv: PublicPath 
    access(contract) let status: {UInt64: String}

    pub event RevealRequest(id: UInt64)
    pub event OpenPackRequest(id: UInt64) 
    pub event Mint(id: UInt64, commitHash: String) 
    pub event ContractInitialized()
    pub event Withdraw(id: UInt64, from: Address?)
    pub event Deposit(id: UInt64, to: Address?)

    /// MintCapReceiver
    ///
    /// This must be linked Publicly so that the PackNFTMinter owner (issuer) can have access to set this
    pub resource interface MintCapReceiver {
        pub fun setMintCap(mintCap: Capability<&PackNFTMinter{IPackNFT.IMinter}>) 
    }
    
    // TODO: remove as capabilities directly shared in PDS.SharedCapabilities
    pub resource PDSMinterProxy: MintCapReceiver {
        access(self) var mintCap:  Capability<&PackNFTMinter{IPackNFT.IMinter}>?
        
        pub fun setMintCap(mintCap: Capability<&PackNFTMinter{IPackNFT.IMinter}>) {
            pre {
                mintCap.borrow() != nil: "Invalid MintCap capability"
            }
            self.mintCap = mintCap
        }

        pub fun mint(commitHash: String, issuer: Address){
           let cap = self.mintCap!.borrow()!
           cap.mint(commitHash: commitHash, issuer: issuer)
        }

        init(){
            self.mintCap = nil
        }
    }

    pub resource PackNFTMinter: IPackNFT.IMinter {
         pub fun mint(commitHash: String, issuer: Address) {
            let id = PackNFT.totalSupply
            let pack <- create NFT(initID: id, commitHash: commitHash, issuer: issuer)
            PackNFT.totalSupply = id + 1
            let recvAcct = getAccount(issuer)
            let recv = recvAcct.getCapability(PackNFT.collectionPublicPath).borrow<&{NonFungibleToken.CollectionPublic}>()
                ?? panic("Unable to borrow Collection Public reference for recipient")

            recv.deposit(token: <- pack)
            log("Minted")
            log(id)
            emit Mint(id: id, commitHash: commitHash)
         }

         init(){}
    }
    
    pub resource NFT: NonFungibleToken.INFT, IPackNFT.IPackNFTToken{
        pub let id: UInt64
        pub let commitHash: String
        pub let issuer: Address

        pub fun reveal(){
            PackNFT.reveal(id: self.id)
        }
        
        pub fun open(){
            PackNFT.open(id: self.id)
        }

        init(initID: UInt64, commitHash: String, issuer: Address ) {
            self.id = initID
            self.commitHash = commitHash
            self.issuer = issuer
        }

    }
    
       pub resource Collection: NonFungibleToken.Provider, NonFungibleToken.Receiver, NonFungibleToken.CollectionPublic {
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

        destroy() {
            destroy self.ownedNFTs
        }
    }
    
    access(contract) fun reveal(id: UInt64) {
        self.status[id] = "Revealed"
        emit RevealRequest(id: id)
    }
    
    access(contract) fun open(id: UInt64) {
        self.status[id] = "Opened"
        emit OpenPackRequest(id: id)
    }
    
    pub fun getStatus(id: UInt64): String? {
        return self.status[id]
    }
    
    pub fun createNewMinterProxy(): @PDSMinterProxy {
        return <- create PDSMinterProxy()
    }

    pub fun createEmptyCollection(): @NonFungibleToken.Collection {
        let c <- create Collection()
        return <- c
    }  
    
    init(
        adminAccount: AuthAccount, 
        collectionStoragePath: StoragePath,
        collectionPublicPath: PublicPath,
        minterStoragePath: StoragePath,
        minterPrivPath: PrivatePath,
        minterProxyStoragePath: StoragePath,
        minterProxyMintCapRecv: PublicPath,
        version: String
    ){
        self.totalSupply = 0
        self.status = {}
        self.collectionStoragePath = collectionStoragePath
        self.collectionPublicPath = collectionPublicPath
        self.minterStoragePath = minterStoragePath
        self.minterPrivPath = minterPrivPath
        self.minterProxyStoragePath = minterProxyStoragePath
        self.minterProxyMintCapRecv = minterProxyMintCapRecv
        self.version = version

        // Create a collection to receive Pack NFTs
        let collection <- create Collection()
        adminAccount.save(<-collection, to: self.collectionStoragePath)
        adminAccount.link<&Collection{NonFungibleToken.CollectionPublic}>(self.collectionPublicPath, target: self.collectionStoragePath)

        // Create a minter to share mint capability with proxy 
        let minter <- create PackNFTMinter()
        adminAccount.save(<-minter, to: self.minterStoragePath)
        adminAccount.link<&PackNFTMinter{IPackNFT.IMinter}>(self.minterPrivPath, target: self.minterStoragePath)
    }

}
 
