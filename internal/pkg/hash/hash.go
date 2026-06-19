package hash

import "golang.org/x/crypto/bcrypt"

const defaultCost = bcrypt.DefaultCost

func Password(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), defaultCost)
	return string(b), err
}

func ComparePassword(hashed, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain)) == nil
}
