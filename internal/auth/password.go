package auth

import (
	"github.com/alexedwards/argon2id"
)

type Hasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) (bool, error)
}

type Argon2Hasher struct {
	params *argon2id.Params
}

func NewArgon2Hasher() *Argon2Hasher {
	return &Argon2Hasher{params: argon2id.DefaultParams}
}

func (h *Argon2Hasher) Hash(password string) (string, error) {
	return argon2id.CreateHash(password, h.params)
}

func (h *Argon2Hasher) Compare(hash, password string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, hash)
}
