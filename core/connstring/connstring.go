package connstring

//go:generate go run spec_connstring_test_generator.go

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/10gen/mongo-go-driver/internal"
)

// Parse parses the provided uri and returns a URI object.
func Parse(s string) (ConnString, error) {
	var p parser
	err := p.parse(s)
	if err != nil {
		err = internal.WrapErrorf(err, "error parsing uri (%s)", s)
	}
	return p.ConnString, err
}

// ConnString represents a connection string to mongodb.
type ConnString struct {
	Original                string
	AuthMechanism           string
	AuthMechanismProperties map[string]string
	Database                string
	Hosts                   []string
	Password                string
	PasswordSet             bool
	ReplicaSet              string
	Username                string
	WTimeout                time.Duration

	UnknownOptions map[string][]string
}

func (u *ConnString) String() string {
	return u.Original
}

type parser struct {
	ConnString

	haveWTimeoutMS bool
}

func (p *parser) parse(original string) error {
	p.Original = original

	uri := original
	var err error
	// scheme
	if !strings.HasPrefix(uri, "mongodb://") {
		return fmt.Errorf("scheme must be \"mongodb\"")
	}

	// user info
	uri = uri[10:]

	if idx := strings.Index(uri, "@"); idx != -1 {
		userInfo := uri[:idx]
		uri = uri[idx+1:]

		username := userInfo
		var password string

		if idx := strings.Index(userInfo, ":"); idx != -1 {
			username = userInfo[:idx]
			password = userInfo[idx+1:]
		}

		p.Username, err = url.QueryUnescape(username)
		if err != nil {
			return internal.WrapErrorf(err, "invalid username")
		}
		if len(password) > 1 {
			if strings.Contains(password, ":") {
				return fmt.Errorf("unescaped colon in password")
			}
			p.Password, err = url.QueryUnescape(password)
			if err != nil {
				return internal.WrapErrorf(err, "invalid password")
			}
			p.PasswordSet = true
		}

	}

	// hosts
	hosts := uri
	if idx := strings.IndexAny(uri, "/?@"); idx != -1 {
		if uri[idx] == '@' {
			return fmt.Errorf("unescaped @ sign in user info")
		}
		if uri[idx] == '?' {
			return fmt.Errorf("must have a / before the query ?")
		}

		hosts = uri[:idx]
	}

	for _, host := range strings.Split(hosts, ",") {
		err = p.addHost(host)
		if err != nil {
			return internal.WrapErrorf(err, "invalid host \"%s\"", host)
		}
	}

	if len(p.Hosts) == 0 {
		return fmt.Errorf("must have at least 1 host")
	}

	uri = uri[len(hosts):]

	if len(uri) == 0 {
		return nil
	}

	if uri[0] != '/' {
		return fmt.Errorf("must have a / separator between hosts and path")
	}

	uri = uri[1:]
	if len(uri) == 0 {
		return nil
	}

	database := uri
	if idx := strings.IndexAny(uri, "?"); idx != -1 {
		database = uri[:idx]
	}

	p.Database, err = url.QueryUnescape(database)
	if err != nil {
		return internal.WrapErrorf(err, "invalid database \"%s\"", database)
	}

	uri = uri[len(database):]

	if len(uri) == 0 {
		return nil
	}

	if uri[0] != '?' {
		return fmt.Errorf("must have a ? separator between path and query")
	}

	uri = uri[1:]
	if len(uri) == 0 {
		return nil
	}

	for _, pair := range strings.FieldsFunc(uri, func(r rune) bool { return r == ';' || r == '&' }) {
		err = p.addOption(pair)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *parser) addHost(host string) error {
	if host == "" {
		return nil
	}
	host, err := url.QueryUnescape(host)
	if err != nil {
		return internal.WrapErrorf(err, "invalid host \"%s\"", host)
	}

	_, port, err := net.SplitHostPort(host)
	// this is unfortunate that SplitHostPort actually requires
	// a port to exist.
	if err != nil {
		if addrError, ok := err.(*net.AddrError); !ok || addrError.Err != "missing port in address" {
			return err
		}
	}

	if port != "" {
		d, err := strconv.Atoi(port)
		if err != nil {
			return internal.WrapErrorf(err, "port must be an integer")
		}
		if d <= 0 || d >= 65536 {
			return fmt.Errorf("port must be in the range [1, 65535]")
		}
	}
	p.Hosts = append(p.Hosts, host)
	return nil
}

func (p *parser) addOption(pair string) error {
	kv := strings.SplitN(pair, "=", 2)
	if len(kv) != 2 || kv[0] == "" {
		return fmt.Errorf("invalid option")
	}

	key, err := url.QueryUnescape(kv[0])
	if err != nil {
		return internal.WrapErrorf(err, "invalid option key \"%s\"", kv[0])
	}

	value, err := url.QueryUnescape(kv[1])
	if err != nil {
		return internal.WrapErrorf(err, "invalid option value \"%s\"", kv[1])
	}

	lowerKey := strings.ToLower(key)
	switch lowerKey {
	case "authmechanism":
		p.AuthMechanism = value
	case "authmechanismproperties":
		p.AuthMechanismProperties = make(map[string]string)
		pairs := strings.Split(value, ",")
		for _, pair := range pairs {
			kv := strings.SplitN(pair, ":", 2)
			if len(kv) != 2 || kv[0] == "" {
				return fmt.Errorf("invalid authMechanism property")
			}
			p.AuthMechanismProperties[kv[0]] = kv[1]
		}
	case "replicaset":
		p.ReplicaSet = value
	case "wtimeoutms":
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid value for wtimeoutMS: %v", value)
		}
		p.WTimeout = time.Duration(n) * time.Millisecond
		p.haveWTimeoutMS = true
	case "wtimeout":
		if p.haveWTimeoutMS {
			// use wtimeoutMS if it exists
			break
		}
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid value for wtimeoutMS: %v", value)
		}
		p.WTimeout = time.Duration(n) * time.Millisecond
	default:
		if p.UnknownOptions == nil {
			p.UnknownOptions = make(map[string][]string)
		}
		p.UnknownOptions[lowerKey] = append(p.UnknownOptions[lowerKey], value)
	}

	return nil
}
