/*
 * Flow Go SDK
 *
 * Copyright 2019 Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cloudkms

import (
	"context"
	"encoding/asn1"
	"fmt"
	"hash/crc32"
	"math/big"

	kms "cloud.google.com/go/kms/apiv1"
	"github.com/onflow/flow-go-sdk/crypto"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var _ crypto.Signer = (*Signer)(nil)

// Signer is a Google Cloud KMS implementation of crypto.Signer.
type Signer struct {
	ctx    context.Context
	client *kms.KeyManagementClient
	key    Key
	// ECDSA is the only algorithm supported by this package. The signature algorithm
	// therefore represents the elliptic curve used. The curve is needed to parse the kms signature.
	curve crypto.SignatureAlgorithm
	// public key for easier access
	publicKey crypto.PublicKey
}

// SignerForKey returns a new Google Cloud KMS signer for an asymmetric signing key version.
//
// Only ECDSA keys on P-256 and secp256k1 curves and SHA2-256 are supported.
func (c *Client) SignerForKey(
	ctx context.Context,
	key Key,
) (*Signer, error) {

	pk, _, err := c.GetPublicKey(ctx, key)
	if err != nil {
		return nil, err
	}

	return &Signer{
		ctx:       ctx,
		client:    c.client,
		key:       key,
		curve:     pk.Algorithm(),
		publicKey: pk,
	}, nil
}

// Sign signs the given message using the KMS signing key for this signer.
//
// Reference: https://cloud.google.com/kms/docs/create-validate-signatures
func (s *Signer) Sign(message []byte) ([]byte, error) {

	request := &kmspb.AsymmetricSignRequest{
		Name:       s.key.ResourceID(),
		Data:       message,
		DataCrc32C: checksum(message),
	}

	result, err := s.client.AsymmetricSign(s.ctx, request)
	if err != nil {
		return nil, fmt.Errorf("cloudkms: failed to sign: %w", err)
	}

	sig, err := parseSignature(result.Signature, s.curve)
	if err != nil {
		return nil, fmt.Errorf("cloudkms: failed to parse signature: %w", err)
	}

	return sig, nil
}

func checksum(data []byte) *wrapperspb.Int64Value {
	// compute the checksum
	checksum := crc32.ChecksumIEEE(data)
	val := wrapperspb.Int64(int64(checksum))
	return val
}

// parseSignature parses an asn1 stucture (R,S) into a slice of bytes as required by the `Siger.Sign` method.
func parseSignature(kmsSignature []byte, curve crypto.SignatureAlgorithm) ([]byte, error) {
	var parsedSig struct{ R, S *big.Int }
	if _, err := asn1.Unmarshal(kmsSignature, &parsedSig); err != nil {
		return nil, fmt.Errorf("asn1.Unmarshal: %w", err)
	}

	curveOrderLen := curveOrder(curve)
	signature := make([]byte, 2*curveOrderLen)

	// left pad R and S with zeroes
	rBytes := parsedSig.R.Bytes()
	sBytes := parsedSig.S.Bytes()
	copy(signature[curveOrderLen-len(rBytes):], rBytes)
	copy(signature[len(signature)-len(sBytes):], sBytes)

	return signature, nil
}

// returns the curve order size in bytes (used to padd R and S of the ECDSA signature)
// Only P-256 and secp256k1 are supported. The calling function should make sure
// the function is only called with one of the 2 curves.
func curveOrder(curve crypto.SignatureAlgorithm) int {
	switch curve {
	case crypto.ECDSA_P256:
		return 32
	case crypto.ECDSA_secp256k1:
		return 32
	default:
		return 0 // or panic? this only happens if there is an implementation bug
	}
}

func (s *Signer) PublicKey() crypto.PublicKey {
	return s.publicKey
}
