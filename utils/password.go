package utils

import (
	"fmt"
	"math/rand"
	"time"
)

type PasswordGenerator struct {
	Passwd              bool
	DisabledDigital     bool
	DisabledLowercase   bool
	DisabledUppercase   bool
	DisabledPunctuation bool
	PasswordLength      int
}

func (pg *PasswordGenerator) GenPassword() error {
	charset := []byte("")
	rand.Seed(time.Now().UnixNano())
	if !pg.DisabledDigital {
		for i := 0; i <= rand.Intn(10); i++ {
			charset = append(charset, []byte("0123456789")...)
		}
	}
	if !pg.DisabledLowercase {
		for i := 0; i <= rand.Intn(10); i++ {
			charset = append(charset, []byte("abcdefghijklmnopqrstuvwxyz")...)
		}
	}
	if !pg.DisabledUppercase {
		for i := 0; i <= rand.Intn(10); i++ {
			charset = append(charset, []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ")...)
		}
	}
	if !pg.DisabledPunctuation {
		charset = append(charset, []byte("!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~")...)
	}

	if len(charset) == 0 {
		return fmt.Errorf("Charset empty!")
	}

	rand.Shuffle(len(charset), func(i, j int) {
		charset[i], charset[j] = charset[j], charset[i]
	})
	passwd := make([]byte, pg.PasswordLength)
	for i := 0; i < pg.PasswordLength; i++ {
		passwd[i] = charset[rand.Intn(len(charset))]
	}
	fmt.Printf("Password: %#v\n", string(passwd))
	return nil
}
