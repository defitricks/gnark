// Copyright 2020 ConsenSys Software Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package eddsa

import (
	"encoding/pem"
	"errors"
	"hash"
	"io"
	"io/ioutil"
	"math/big"

	"github.com/consensys/gurvy/bn256/twistededwards"
	"golang.org/x/crypto/blake2b"
)

var errNotOnCurve = errors.New("point not on curve")

const (
	frSize         = 32 // TODO assumes a 256 bits field for the twisted curve (ok for our implem)
	sizePublicKey  = 2 * frSize
	sizeSignature  = 3 * frSize
	sizePrivateKey = 4 * frSize
	keyType        = "ED JUBJUB BN256 PUBLIC KEY"
)

// PublicKey eddsa signature object
// cf https://en.wikipedia.org/wiki/EdDSA for notation
type PublicKey struct {
	A twistededwards.PointAffine
}

// SetBytes sets p from binary representation in buf.
// buf represents a public key as x||y where x, y are
// interpreted as big endian binary numbers corresponding
// to the coordinates of a point on the twisted Edwards.
// It returns the number of bytes read from the buffer.
func (pk *PublicKey) SetBytes(buf []byte) (int, error) {
	n := 0
	if len(buf) < sizePublicKey {
		return n, io.ErrShortBuffer
	}
	pk.A.X.SetBytes(buf[:frSize])
	pk.A.Y.SetBytes(buf[frSize : 2*frSize])
	n += 2 * frSize
	if !pk.A.IsOnCurve() {
		return n, errNotOnCurve
	}
	return n, nil
}

// Unmarshal alis to SetBytes.
func (pk *PublicKey) Unmarshal(buf []byte) error {
	_, err := pk.SetBytes(buf)
	return err
}

// Bytes returns the binary representation of pk
// as x||y where x, y are the coordinates of the point
// on the twisted Edwards as big endian integers.
func (pk *PublicKey) Bytes() [sizePublicKey]byte {
	var res [sizePublicKey]byte
	x := pk.A.X.Bytes()
	y := pk.A.Y.Bytes()
	copy(res[:], x[:])
	copy(res[frSize:], y[:])
	return res
}

// Marshal converts pk to binary, returning it as
// a byte slice.
func (pk *PublicKey) Marshal() []byte {
	b := pk.Bytes()
	return b[:]
}

// DumpToPEM writes the content of pk to a PEM file
// named s.
func (pk *PublicKey) DumpToPEM(s string) error {
	b := pk.Bytes()
	block := &pem.Block{
		Type:  keyType,
		Bytes: b[:],
	}

	err := ioutil.WriteFile(s, pem.EncodeToMemory(block), 0x0600)
	return err
}

// Signature represents an eddsa signature
// cf https://en.wikipedia.org/wiki/EdDSA for notation
type Signature struct {
	R twistededwards.PointAffine
	S [frSize]byte
}

// SetBytes sets sig from a buffer in binary.
// buf is read interpreted as x||y||s where
// * x,y are the coordinates of a point on the twisted
//	Edwards represented in big endian
// * s=r+h(r,a,m) mod l, the Hasse bound guarantess that
//	s is smaller than frSize (in particular it is supposed
// 	s is NOT blinded)
// It returns the number of bytes read from buf.
func (sig *Signature) SetBytes(buf []byte) (int, error) {
	n := 0
	if len(buf) < sizeSignature {
		return n, io.ErrShortBuffer
	}
	sig.R.X.SetBytes(buf[:frSize])
	sig.R.Y.SetBytes(buf[frSize : 2*frSize])
	n += 2 * frSize
	if !sig.R.IsOnCurve() {
		return n, errNotOnCurve
	}
	copy(sig.S[:], buf[2*frSize:3*frSize])
	n += frSize
	return n, nil
}

// Unmarshal alias to SetBytes.
func (sig *Signature) Unmarshal(buf []byte) error {
	_, err := sig.SetBytes(buf)
	return err
}

// Bytes returns the binary representation of sig
// as a byte array of size 3*frSize x||y||s where
// * x, y are the coordinates of a point on the twisted
//	Edwards represented in big endian
// * s=r+h(r,a,m) mod l, the Hasse bound guarantess that
//	s is smaller than frSize (in particular it is supposed
// 	s is NOT blinded)
func (sig *Signature) Bytes() [sizeSignature]byte {
	var res [sizeSignature]byte
	x := sig.R.X.Bytes()
	y := sig.R.Y.Bytes()
	copy(res[:frSize], x[:])
	copy(res[frSize:], y[:])
	copy(res[2*frSize:], sig.S[:])
	return res
}

// Marshal converts pk to binary, returning it as
// a byte slice.
func (sig *Signature) Marshal() []byte {
	b := sig.Bytes()
	return b[:]
}

// PrivateKey private key of an eddsa instance
type PrivateKey struct {
	pubKey  PublicKey    // copy of the associated public key
	scalar  [frSize]byte // secret scalar, in big Endian
	randSrc [32]byte     // randomizer (non need to convert it when doing scalar mul --> random = H(randSrc,msg))
}

// SetBytes sets pk from buf, where buf is interpreted
// as  publicKey||scalar||randSrc
// where publicKey is as publicKey.Bytes(), and
// scalar is in big endian, of size frSize.
// It returns the number byte read.
func (privKey *PrivateKey) SetBytes(buf []byte) (int, error) {
	n := 0
	if len(buf) < sizePrivateKey {
		return n, io.ErrShortBuffer
	}
	privKey.pubKey.A.X.SetBytes(buf[:frSize])
	privKey.pubKey.A.Y.SetBytes(buf[frSize : 2*frSize])
	n += 2 * frSize
	if !privKey.pubKey.A.IsOnCurve() {
		return n, errNotOnCurve
	}
	copy(privKey.scalar[:], buf[2*frSize:3*frSize])
	copy(privKey.randSrc[:], buf[3*frSize:])
	n += frSize
	return n, nil
}

// Unmarshal alias to SetBytes.
func (privKey *PrivateKey) Unmarshal(buf []byte) error {
	_, err := privKey.SetBytes(buf)
	return err
}

// Bytes returns the binary representation of pk,
// as byte array publicKey||scalar||randSrc
// where publicKey is as publicKey.Bytes(), and
// scalar is in big endian, of size frSize.
func (privKey *PrivateKey) Bytes() [sizePrivateKey]byte {
	var res [sizePrivateKey]byte
	x := privKey.pubKey.A.X.Bytes()
	y := privKey.pubKey.A.Y.Bytes()
	copy(res[:], x[:])
	copy(res[frSize:], y[:])
	copy(res[2*frSize:], privKey.scalar[:])
	copy(res[3*frSize:], privKey.randSrc[:])
	return res
}

// Marshal converts privKey to binary, returning it as
// a byte slice.
func (privKey *PrivateKey) Marshal() []byte {
	b := privKey.Bytes()
	return b[:]
}

// New creates an instance of eddsa
func New(seed [32]byte) (PublicKey, PrivateKey) {

	c := twistededwards.GetEdwardsCurve()

	var pub PublicKey
	var priv PrivateKey

	h := blake2b.Sum512(seed[:])
	for i := 0; i < 32; i++ {
		priv.randSrc[i] = h[i+32]
	}

	// prune the key
	// https://tools.ietf.org/html/rfc8032#section-5.1.5, key generation
	h[0] &= 0xF8
	h[31] &= 0x7F
	h[31] |= 0x40

	// reverse first bytes because setBytes interpret stream as big endian
	// but in eddsa specs s is the first 32 bytes in little endian
	for i, j := 0, 32; i < j; i, j = i+1, j-1 {
		h[i], h[j] = h[j], h[i]
	}

	copy(priv.scalar[:], h[:32])

	var bscalar big.Int
	bscalar.SetBytes(priv.scalar[:])
	pub.A.ScalarMul(&c.Base, &bscalar)

	priv.pubKey = pub

	return pub, priv
}

// Sign sign a message
// cf https://en.wikipedia.org/wiki/EdDSA for the notations
// Eddsa is supposed to be built upon Edwards (or twisted Edwards) curves having 256 bits group size and cofactor=4 or 8
func Sign(message []byte, priv PrivateKey, hFunc hash.Hash) (Signature, error) {

	curveParams := twistededwards.GetEdwardsCurve()

	var res Signature

	var randScalarInt big.Int

	// randSrc = privKey.randSrc || msg (-> message = MSB message .. LSB message)
	randSrc := make([]byte, 32+len(message))
	for i, v := range priv.randSrc {
		randSrc[i] = v
	}
	copy(randSrc[32:], message)

	// randBytes = H(randSrc)
	randBytes := blake2b.Sum512(randSrc[:]) // TODO ensures that the hash used to build the key and the one used here is the same
	randScalarInt.SetBytes(randBytes[:32])

	// compute R = randScalar*Base
	res.R.ScalarMul(&curveParams.Base, &randScalarInt)
	if !res.R.IsOnCurve() {
		return Signature{}, errNotOnCurve
	}

	// compute H(R, A, M), all parameters in data are in Montgomery form
	resRX := res.R.X.Bytes()
	resRY := res.R.Y.Bytes()
	resAX := priv.pubKey.A.X.Bytes()
	resAY := priv.pubKey.A.Y.Bytes()
	sizeDataToHash := 4*frSize + len(message)
	dataToHash := make([]byte, sizeDataToHash)
	copy(dataToHash[:], resRX[:])
	copy(dataToHash[frSize:], resRY[:])
	copy(dataToHash[2*frSize:], resAX[:])
	copy(dataToHash[3*frSize:], resAY[:])
	copy(dataToHash[4*frSize:], message)
	hFunc.Reset()
	_, err := hFunc.Write(dataToHash[:])
	if err != nil {
		return Signature{}, err
	}

	var hramInt big.Int
	hramBin := hFunc.Sum([]byte{})
	hramInt.SetBytes(hramBin)

	// Compute s = randScalarInt + H(R,A,M)*S
	// going with big int to do ops mod curve order
	var bscalar, bs big.Int
	bscalar.SetBytes(priv.scalar[:])
	bs.Mul(&hramInt, &bscalar).
		Add(&bs, &randScalarInt).
		Mod(&bs, &curveParams.Order)
	sb := bs.Bytes()
	if len(sb) < frSize {
		offset := make([]byte, frSize-len(sb))
		sb = append(offset, sb...)
	}
	copy(res.S[:], sb[:])

	return res, nil
}

// Verify verifies an eddsa signature
// cf https://en.wikipedia.org/wiki/EdDSA
func Verify(sig Signature, message []byte, pub PublicKey, hFunc hash.Hash) (bool, error) {

	curveParams := twistededwards.GetEdwardsCurve()

	// verify that pubKey and R are on the curve
	if !pub.A.IsOnCurve() {
		return false, errNotOnCurve
	}

	// compute H(R, A, M), all parameters in data are in Montgomery form
	sigRX := sig.R.X.Bytes()
	sigRY := sig.R.Y.Bytes()
	sigAX := pub.A.X.Bytes()
	sigAY := pub.A.Y.Bytes()
	sizeDataToHash := 4*frSize + len(message)
	dataToHash := make([]byte, sizeDataToHash)
	copy(dataToHash[:], sigRX[:])
	copy(dataToHash[frSize:], sigRY[:])
	copy(dataToHash[2*frSize:], sigAX[:])
	copy(dataToHash[3*frSize:], sigAY[:])
	copy(dataToHash[4*frSize:], message)
	hFunc.Reset()
	_, err := hFunc.Write(dataToHash[:])
	if err != nil {
		return false, err
	}

	var hramInt big.Int
	hramBin := hFunc.Sum([]byte{})
	hramInt.SetBytes(hramBin)

	// lhs = cofactor*S*Base
	var lhs twistededwards.PointAffine
	var bCofactor, bs big.Int
	curveParams.Cofactor.ToBigInt(&bCofactor)
	bs.SetBytes(sig.S[:])
	lhs.ScalarMul(&curveParams.Base, &bs).
		ScalarMul(&lhs, &bCofactor)

	if !lhs.IsOnCurve() {
		return false, errNotOnCurve
	}

	// rhs = cofactor*(R + H(R,A,M)*A)
	var rhs twistededwards.PointAffine
	rhs.ScalarMul(&pub.A, &hramInt).
		Add(&rhs, &sig.R).
		ScalarMul(&rhs, &bCofactor)
	if !rhs.IsOnCurve() {
		return false, errNotOnCurve
	}

	// verifies that cofactor*S*Base=cofactor*(R + H(R,A,M)*A)
	if !lhs.X.Equal(&rhs.X) || !lhs.Y.Equal(&rhs.Y) {
		return false, nil
	}
	return true, nil
}
