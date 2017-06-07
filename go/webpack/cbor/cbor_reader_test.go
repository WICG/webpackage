package cbor_test

import (
	"testing"

	"github.com/dimich-g/webpackage/go/webpack/cbor"
	"github.com/stretchr/testify/assert"
)

func decode(hex string) (cbor.Type, uint64, int, error) {
	d := cbor.NewDecoder(fromHex(hex))
	typ, value, err := d.Decode()
	return typ, value, d.Pos, err
}

func TestDecoded(t *testing.T) {
	assert := assert.New(t)
	typ, value, err := cbor.NewDecoder(nil).Decode()
	assert.Error(err)
	typ, value, pos, err := decode("")
	assert.Error(err)
	typ, value, pos, err = decode("01")
	assert.NoError(err)
	assert.Equal(cbor.TypePosInt, typ)
	assert.EqualValues(1, value)
	assert.Equal(1, pos)

	typ, value, pos, err = decode("18")
	assert.Error(err)
	assert.Equal(0, pos)

	typ, value, pos, err = decode("1842")
	assert.NoError(err)
	assert.Equal(2, pos)
	assert.Equal(cbor.TypePosInt, typ)
	assert.EqualValues(0x42, value)

	typ, value, pos, err = decode("184243")
	assert.NoError(err)
	assert.Equal(2, pos)
	assert.Equal(cbor.TypePosInt, typ)
	assert.EqualValues(0x42, value)

	typ, value, pos, err = decode("19010203")
	assert.NoError(err)
	assert.Equal(3, pos)
	assert.Equal(cbor.TypePosInt, typ)
	assert.EqualValues(0x0102, value)

	typ, value, pos, err = decode("1a010203040506")
	assert.NoError(err)
	assert.Equal(5, pos)
	assert.Equal(cbor.TypePosInt, typ)
	assert.EqualValues(0x01020304, value)

	typ, value, pos, err = decode("1b010203040506070809")
	assert.NoError(err)
	assert.Equal(9, pos)
	assert.Equal(cbor.TypePosInt, typ)
	assert.EqualValues(0x0102030405060708, value)

	typ, value, pos, err = decode("1c0102030405060708090a0b0c0d0e0f1011")
	assert.Error(err)
	assert.Equal(0, pos)

	typ, value, pos, err = decode("1f0102030405060708090a0b0c0d0e0f1011")
	assert.Error(err)
	assert.Equal(0, pos)

	typ, value, pos, err = decode("61")
	assert.NoError(err)
	assert.Equal(1, pos)
	assert.Equal(cbor.TypeText, typ)
	assert.EqualValues(1, value)
}
