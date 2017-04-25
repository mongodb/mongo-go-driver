//+build gssapi
//+build linux darwin freebsd

package gssapi

/*
#cgo linux LDFLAGS: -lgssapi_krb5 -lkrb5
#cgo freebsd pkg-config: heimdal-gssapi
#include "gssapi_wrapper.h"
*/
import "C"
import (
	"fmt"
	"net"
	"runtime"
	"strings"
	"unsafe"
)

// New creates a new SaslClient.
func New(target, username, password string, passwordSet bool, props map[string]string) (*SaslClient, error) {
	if passwordSet {
		return nil, fmt.Errorf("passwords are not supported when using gssapi on %s", runtime.GOOS)
	}
	serviceName := "mongodb"

	for key, value := range props {
		switch strings.ToUpper(key) {
		case "CANONICALIZE_HOST_NAME":
			return nil, fmt.Errorf("CANONICALIZE_HOST_NAME is not supported when using gssapi on %s", runtime.GOOS)
		case "SERVICE_REALM":
			return nil, fmt.Errorf("SERVICE_REALM is not supported when using gssapi on %s", runtime.GOOS)
		case "SERVICE_NAME":
			serviceName = value
		default:
			return nil, fmt.Errorf("unknown mechanism property %s", key)
		}
	}

	hostname, _, err := net.SplitHostPort(target)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint (%s) specified: %s", target, err)
	}

	servicePrincipalName := fmt.Sprintf("%s@%s", serviceName, hostname)

	return &SaslClient{
		servicePrincipalName: servicePrincipalName,
		username:             username,
	}, nil
}

type SaslClient struct {
	servicePrincipalName string
	username             string

	// state
	state           C.gssapi_client_state
	contextComplete bool
	done            bool
}

func (sc *SaslClient) Close() {
	C.gssapi_client_destroy(&sc.state)
}

func (sc *SaslClient) Start() (string, []byte, error) {
	const mechName = "GSSAPI"

	cservicePrincipalName := C.CString(sc.servicePrincipalName)
	defer C.free(unsafe.Pointer(cservicePrincipalName))
	var cusername *C.char
	if sc.username != "" {
		cusername := C.CString(sc.username)
		defer C.free(unsafe.Pointer(cusername))
	}
	status := C.gssapi_client_init(&sc.state, cservicePrincipalName, cusername)

	if status != C.GSSAPI_OK {
		return mechName, nil, fmt.Errorf("unable to initialize gssapi client state: %v,%v", sc.state.maj_stat, sc.state.min_stat)
	}

	return mechName, nil, nil
}

func (sc *SaslClient) Next(challenge []byte) ([]byte, error) {

	var outBuf unsafe.Pointer
	var outBufLen C.size_t

	if sc.contextComplete {
		if sc.username == "" {
			var cusername *C.char
			status := C.gssapi_client_get_username(&sc.state, &cusername)
			if status != C.GSSAPI_OK {
				return nil, fmt.Errorf("unable to acquire username: %v,%v", int32(sc.state.maj_stat), int32(sc.state.min_stat))
			}
			defer C.free(unsafe.Pointer(cusername))
			sc.username = C.GoString((*C.char)(unsafe.Pointer(cusername)))
		}

		bytes := append([]byte{1, 0, 0, 0}, []byte(sc.username)...)
		buf := unsafe.Pointer(&bytes[0])
		bufLen := C.size_t(len(bytes))
		status := C.gssapi_client_wrap_msg(&sc.state, buf, bufLen, &outBuf, &outBufLen)
		if status != C.GSSAPI_OK {
			return nil, fmt.Errorf("unable to wrap authz: %v,%v", int32(sc.state.maj_stat), int32(sc.state.min_stat))
		}

		sc.done = true
	} else {
		var buf unsafe.Pointer
		var bufLen C.size_t
		if len(challenge) > 0 {
			buf = unsafe.Pointer(&challenge[0])
			bufLen = C.size_t(len(challenge))
		}

		status := C.gssapi_init_sec_context(&sc.state, buf, bufLen, &outBuf, &outBufLen)
		switch status {
		case C.GSSAPI_OK:
			sc.contextComplete = true
		case C.GSSAPI_CONTINUE:
		default:
			return nil, fmt.Errorf("unable to initialize sec context: %v,%v", int32(sc.state.maj_stat), int32(sc.state.min_stat))
		}
	}

	if outBuf != nil {
		defer C.free(outBuf)
	}

	return C.GoBytes(outBuf, C.int(outBufLen)), nil
}

func (sc *SaslClient) Completed() bool {
	return sc.done
}
