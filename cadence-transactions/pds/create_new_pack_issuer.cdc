import PDS from 0x{{.PDS}}

transaction() {
    prepare (issuer: AuthAccount) {
        
        // Check if account already have a PackIssuer resource, if so destroy it
        if issuer.borrow<&PDS.PackIssuer>(from: PDS.packIssuerStoragePath) != nil {
            issuer.unlink(PDS.packIssuerCapRecv)
            let p <- issuer.load<@PDS.PackIssuer>(from: PDS.packIssuerStoragePath) 
            destroy p
        }
        
        issuer.save(<- PDS.createPackIssuer(), to: PDS.packIssuerStoragePath);
        
        issuer.link<&PDS.PackIssuer{PDS.PackIssuerCapReciever}>(PDS.packIssuerCapRecv, target: PDS.packIssuerStoragePath)
        ??  panic("Could not link packIssuerCapReceiver");
    } 
}
 
