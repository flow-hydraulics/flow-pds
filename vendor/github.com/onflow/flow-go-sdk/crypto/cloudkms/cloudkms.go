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

// Package cloudkms provides a Google Cloud Key Management Service (KMS)
// implementation of the crypto.Signer interface.
//
// The documentation for Google Cloud KMS can be found here: https://cloud.google.com/kms/docs
package cloudkms

import (
	"context"
	"fmt"
	"strings"

	kms "cloud.google.com/go/kms/apiv1"
	"google.golang.org/api/option"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"

	"github.com/onflow/flow-go-sdk/crypto"
)

const (
	resourceIDFormat        = "projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s/cryptoKeyVersions/%s"
	resourceIDArgumentCount = 5
)

// Key is a reference to a Google Cloud KMS asymmetric signing key version.
//
// Ref: https://cloud.google.com/kms/docs/creating-asymmetric-keys#create_an_asymmetric_signing_key
type Key struct {
	ProjectID  string `json:"projectId"`
	LocationID string `json:"locationId"`
	KeyRingID  string `json:"keyRingId"`
	KeyID      string `json:"keyId"`
	KeyVersion string `json:"keyVersion"`
}

// ResourceID returns the resource ID for this KMS key version.
//
// Ref: https://cloud.google.com/kms/docs/getting-resource-ids
func (k Key) ResourceID() string {
	return fmt.Sprintf(
		resourceIDFormat,
		k.ProjectID,
		k.LocationID,
		k.KeyRingID,
		k.KeyID,
		k.KeyVersion,
	)
}

// KeyFromResourceID returns a `Key` from a resource ID.
func KeyFromResourceID(resourceID string) (Key, error) {
	key := Key{}

	scanned, err := fmt.Sscanf(
		strings.ReplaceAll(resourceID, "/", " "),       // input
		strings.ReplaceAll(resourceIDFormat, "/", " "), // format
		&key.ProjectID, &key.LocationID, &key.KeyRingID, &key.KeyID, &key.KeyVersion, // arguments to fill
	)
	if err != nil {
		return key, fmt.Errorf("cloudkms: failed to parse resource ID %s, scanf error: %w", resourceID, err)
	}

	if scanned != resourceIDArgumentCount {
		return key, fmt.Errorf("cloudkms: failed to parse resource ID %s, found %d arguments but expected %d", resourceID, scanned, resourceIDArgumentCount)
	}

	return key, nil
}

// Client is a client for interacting with the Google Cloud KMS API
// using types native to the Flow Go SDK.
type Client struct {
	client *kms.KeyManagementClient
}

// NewClient creates a new KMS client.
func NewClient(ctx context.Context, opts ...option.ClientOption) (*Client, error) {
	client, err := kms.NewKeyManagementClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("cloudkms: failed to initialize client: %w", err)
	}

	return &Client{
		client: client,
	}, nil
}

// GetPublicKey fetches the public key portion of a KMS asymmetric signing key version.
//
// KMS keys of the type `CryptoKeyVersion_EC_SIGN_P256_SHA256` and `CryptoKeyVersion_EC_SIGN_SECP256K1_SHA256`
// are the only keys supported by the SDK.
//
// Ref: https://cloud.google.com/kms/docs/retrieve-public-key
func (c *Client) GetPublicKey(ctx context.Context, key Key) (crypto.PublicKey, crypto.HashAlgorithm, error) {
	request := &kmspb.GetPublicKeyRequest{
		Name: key.ResourceID(),
	}

	result, err := c.client.GetPublicKey(ctx, request)
	if err != nil {
		return nil,
			crypto.UnknownHashAlgorithm,
			fmt.Errorf("cloudkms: failed to fetch public key from KMS API: %v", err)
	}

	sigAlgo := ParseSignatureAlgorithm(result.Algorithm)
	if sigAlgo == crypto.UnknownSignatureAlgorithm {
		return nil,
			crypto.UnknownHashAlgorithm,
			fmt.Errorf(
				"cloudkms: unsupported signature algorithm %s",
				result.Algorithm.String(),
			)
	}

	hashAlgo := ParseHashAlgorithm(result.Algorithm)
	if hashAlgo == crypto.UnknownHashAlgorithm {
		return nil,
			crypto.UnknownHashAlgorithm,
			fmt.Errorf(
				"cloudkms: unsupported hash algorithm %s",
				result.Algorithm.String(),
			)
	}

	publicKey, err := crypto.DecodePublicKeyPEM(sigAlgo, result.Pem)
	if err != nil {
		return nil,
			crypto.UnknownHashAlgorithm,
			fmt.Errorf("cryptokms: failed to parse PEM public key: %w", err)
	}

	return publicKey, hashAlgo, nil
}

// KMSClient gives access to the KeyManagementClient,
// e.g. for closing the connection to the Google KMS API
func (c *Client) KMSClient() *kms.KeyManagementClient {
	return c.client
}

// ParseSignatureAlgorithm returns the `SignatureAlgorithm` corresponding to the input KMS key type.
func ParseSignatureAlgorithm(algo kmspb.CryptoKeyVersion_CryptoKeyVersionAlgorithm) crypto.SignatureAlgorithm {
	if algo == kmspb.CryptoKeyVersion_EC_SIGN_P256_SHA256 {
		return crypto.ECDSA_P256
	}

	if algo == kmspb.CryptoKeyVersion_EC_SIGN_SECP256K1_SHA256 {
		return crypto.ECDSA_secp256k1
	}

	return crypto.UnknownSignatureAlgorithm
}

// ParseHashAlgorithm returns the `HashAlgorithm` corresponding to the input KMS key type.
func ParseHashAlgorithm(algo kmspb.CryptoKeyVersion_CryptoKeyVersionAlgorithm) crypto.HashAlgorithm {
	if algo == kmspb.CryptoKeyVersion_EC_SIGN_P256_SHA256 || algo == kmspb.CryptoKeyVersion_EC_SIGN_SECP256K1_SHA256 {
		return crypto.SHA2_256
	}

	// the function can be extended to return SHA3-256 if it becomes supported by KMS.
	return crypto.UnknownHashAlgorithm
}
