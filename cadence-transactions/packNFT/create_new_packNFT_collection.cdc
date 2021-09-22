import PackNFT from 0x{{.PackNFT}}
import NonFungibleToken from 0x{{.NonFungibleToken}}

transaction() {
    prepare (issuer: AuthAccount) {
        
        // Check if account already have a PackIssuer resource, if so destroy it
        if issuer.borrow<&PackNFT.Collection>(from: PackNFT.collectionStoragePath) != nil {
            issuer.unlink(PackNFT.collectionPublicPath)
            let p <- issuer.load<@PackNFT.Collection>(from: PackNFT.collectionStoragePath) 
            destroy p
        }
        
        issuer.save(<- PackNFT.createEmptyCollection(), to: PackNFT.collectionStoragePath);
        
        issuer.link<&{NonFungibleToken.CollectionPublic}>(PackNFT.collectionPublicPath, target: PackNFT.collectionStoragePath)
        ??  panic("Could not link Collection Pub Path");
    } 
}
 
