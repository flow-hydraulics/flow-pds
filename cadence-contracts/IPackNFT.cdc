import Crypto
import NonFungibleToken from "./NonFungibleToken.cdc"


pub contract interface IPackNFT{
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
    pub event Mint(distId: UInt64, id: UInt64, commitHash: String) 
    /// Revealed
    /// 
    /// Emitted when a packNFT has been revealed
    pub event Revealed(id: UInt64, nfts: String, salt: String)
    /// Opened
    ///
    /// Emitted when a packNFT has been opened
    pub event Opened(id: UInt64)

    access(contract) fun revealRequest(id: UInt64)
    access(contract) fun openRequest(id: UInt64)

    pub struct interface Collectible {
        pub let address: Address
        pub let contractName: String
        pub let id: UInt64
        pub fun hashString(): String 
        init(address: Address, contractName: String, id: UInt64)
    }

    // TODO Pack resource
    
    pub resource interface IOperator {
        pub fun mint(commitHash: String, issuer: Address)
        pub fun reveal(id: UInt64, nfts: [{Collectible}], salt: String)
        pub fun open(id: UInt64) 
    }
    pub resource PackNFTOperator: IOperator {
        pub fun mint(commitHash: String, issuer: Address)
        pub fun reveal(id: UInt64, nfts: [{Collectible}], salt: String)
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
