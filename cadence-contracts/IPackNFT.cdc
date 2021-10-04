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
    pub event Mint(id: UInt64, commitHash: String, distId: UInt64 ) 
    /// Revealed
    /// 
    /// Emitted when a packNFT has been revealed
    pub event Revealed(id: UInt64, salt: String, nfts: String)
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
        pub fun mint(distId: UInt64, commitHash: String, issuer: Address): @NFT
        pub fun reveal(id: UInt64, nfts: [{Collectible}], salt: String)
        pub fun open(id: UInt64) 
    }
    pub resource PackNFTOperator: IOperator {
        pub fun mint(distId: UInt64, commitHash: String, issuer: Address): @NFT
        pub fun reveal(id: UInt64, nfts: [{Collectible}], salt: String)
        pub fun open(id: UInt64) 
    }

    pub resource interface IPackNFTToken {
        pub let id: UInt64
        pub let commitHash: String
        pub let issuer: Address
    }

    pub resource NFT: NonFungibleToken.INFT, IPackNFTToken, IPackNFTOwnerOperator{
        pub let id: UInt64
        pub let commitHash: String
        pub let issuer: Address
        pub fun reveal()
        pub fun open() 
    }
    
    pub resource interface IPackNFTOwnerOperator{
        pub fun reveal()
        pub fun open() 
    }
    
    pub resource interface IPackNFTCollectionPublic {
        pub fun deposit(token: @NonFungibleToken.NFT)
        pub fun getIDs(): [UInt64]
        pub fun borrowNFT(id: UInt64): &NonFungibleToken.NFT
        pub fun borrowPackNFT(id: UInt64): &IPackNFT.NFT? {
            // If the result isn't nil, the id of the returned reference
            // should be the same as the argument to the function
            post {
                (result == nil) || (result!.id == id):
                    "Cannot borrow PackNFT reference: The ID of the returned reference is incorrect"
            }
        }
    }
}
