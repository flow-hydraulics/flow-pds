import Crypto
import NonFungibleToken from "./NonFungibleToken.cdc"


pub contract interface IPackNFT{
    /// Status of the PackNFTs
    /// 
    /// map of pack Id : Status {"Sealed", "Revealed", "Opened"}
    access(contract) let status: {UInt64: String}
    /// Content of the PackNFTs
    ///
    ///  map of pack Id : nftIds 
    access(contract) let content: {UInt64: [UInt64]}
    /// StoragePath for Collection Resource
    /// 
    pub let collectionStoragePath: StoragePath 
    /// PublicPath expected for deposit
    /// 
    pub let collectionPublicPath: PublicPath 
    /// Request for Reveal
    ///
    pub event RevealRequest(id: UInt64)
    /// Request for Open
    ///
    /// This is emitted when owner of a PackNFT request for the entitled NFT to be
    /// deposited to its account
    pub event OpenRequest(id: UInt64) 
    /// New Pack NFT
    ///
    /// Emitted when a new PackNFT has been minted
    pub event Mint(id: UInt64, commitHash: String) 
    /// Revealed
    /// 
    /// Emitted when a packNFT has been revealed
    pub event Revealed(id: UInt64, nftIds: [UInt64])
    /// Opened
    ///
    /// Emitted when a packNFT has been opened
    pub event Opened(id: UInt64)

    /// Public function to get status
    pub fun getStatus(id: UInt64): String?

    access(contract) fun reveal(id: UInt64, nftIds: [UInt64]) {
        pre {
            self.status[id] == "Sealed": "PackNFT not sealed"
        }
        post {
            self.status[id] == "Revealed": "PackNFT status must be Revealed"
            self.content.containsKey(id) : "Revealed data must be recorded"
        }
    }
    
    access(contract) fun open(id: UInt64) {
        pre {
            self.status[id] == "Revealed": "PackNFT not yet revealed"
        }
        post {
            self.status[id] == "Opened": "PackNFT status must be Opened"
        }
        
    }
    
    pub resource interface IOperator {
        pub fun mint(commitHash: String, issuer: Address)
        pub fun reveal(id: UInt64, nftIds: [UInt64])
        pub fun open(id: UInt64) 
    }
    pub resource PackNFTOperator: IOperator {
        pub fun mint(commitHash: String, issuer: Address)
        pub fun reveal(id: UInt64, nftIds: [UInt64])
        pub fun open(id: UInt64) 
    }

    pub resource interface IPackNFTToken {
        pub let id: UInt64
        pub let commitHash: String
        pub let issuer: Address
    }

    pub resource NFT: NonFungibleToken.INFT, IPackNFTToken {
        pub let id: UInt64
        pub let commitHash: String
        pub let issuer: Address
    }
    
    pub resource interface IPackNFTOwnerOperator{
        pub fun reveal()
        pub fun open() 
    }
    
    pub resource interface IPackNFTCollection {
        pub fun borrowPackNFT(id: UInt64): &NFT
    }
}
