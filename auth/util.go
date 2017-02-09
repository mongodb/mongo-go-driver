package auth

import (
	"crypto/md5"
	"fmt"
	"io"
)

const defaultAuthDB = "admin"

func mongoPasswordDigest(username, password string) string {
	h := md5.New()
	io.WriteString(h, username)
	io.WriteString(h, ":mongo:")
	io.WriteString(h, password)
	return fmt.Sprintf("%x", h.Sum(nil))
}
