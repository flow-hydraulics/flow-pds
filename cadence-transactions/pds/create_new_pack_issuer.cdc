import PDS from 0x{{.PDS}}

transaction() {
    prepare (issuer: AuthAccount) {
        
        // Check if account already have a PackIssuer resource, if so destroy it
        if issuer.borrow<&PDS.PackIssuer>(from: PDS.PackIssuerStoragePath) != nil {
            issuer.unlink(PDS.PackIssuerCapRecv)
            let p <- issuer.load<@PDS.PackIssuer>(from: PDS.PackIssuerStoragePath) 
            destroy p
        }
        
        issuer.save(<- PDS.createPackIssuer(), to: PDS.PackIssuerStoragePath);
        
        issuer.link<&PDS.PackIssuer{PDS.PackIssuerCapReciever}>(PDS.PackIssuerCapRecv, target: PDS.PackIssuerStoragePath)
        ??  panic("Could not link packIssuerCapReceiver");
    } 
}
 
