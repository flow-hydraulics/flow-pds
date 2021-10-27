import NonFungibleToken from 0x{{.NonFungibleToken}}
import {{.CollectibleNFTName}} from 0x{{.CollectibleNFTAddress}}

transaction (NFTProviderPath: PrivatePath) {
    prepare(signer: AuthAccount) {
        // Setup the collection, if not already
        if signer.borrow<&{{.CollectibleNFTName}}.Collection>(from: {{.CollectibleNFTName}}.CollectionStoragePath) == nil {
          // create a new empty collection
          let collection <- {{.CollectibleNFTName}}.createEmptyCollection()

          // save it to the account
          signer.save(<-collection, to: {{.CollectibleNFTName}}.CollectionStoragePath)

          // create a public capability for the collection
          signer.link<&NonFungibleToken.Collection{NonFungibleToken.CollectionPublic}>({{.CollectibleNFTName}}.CollectionPublicPath, target: {{.CollectibleNFTName}}.CollectionStoragePath)
          assert(signer.getCapability<&{NonFungibleToken.CollectionPublic}>({{.CollectibleNFTName}}.CollectionPublicPath).check(), message: "did not link public cap");
        }

        // Link the private withdraw capability, if not already
        if !signer.getCapability<&{NonFungibleToken.Provider}>(NFTProviderPath).check() {
          signer.link<&{NonFungibleToken.Provider}>(NFTProviderPath, target: {{.CollectibleNFTName}}.CollectionStoragePath)
          assert(signer.getCapability<&{NonFungibleToken.Provider}>(NFTProviderPath).check(), message: "did not link withdraw cap");
        }
    }
}
