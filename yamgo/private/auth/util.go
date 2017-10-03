package auth

import (
	"crypto/md5"
	"fmt"
	"io"
)

const defaultAuthDB = "admin"

func mongoPasswordDigest(username, password string) string {
	h := md5.New()
	_, _ = io.WriteString(h, username)
	_, _ = io.WriteString(h, ":mongo:")
	_, _ = io.WriteString(h, password)
	return fmt.Sprintf("%x", h.Sum(nil))
}
