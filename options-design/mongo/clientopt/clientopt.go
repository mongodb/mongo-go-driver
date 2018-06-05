package clientopt

import (
	"context"
	"net"
	"time"

	"github.com/mongodb/mongo-go-driver/options-design/mongo/readconcern"
)

type ContextDialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

type Option interface {
	option()
}

type OptionFunc func(*Client) error

type Client struct{}

type Bundle struct{}

func BundleOptions(...Option) Bundle { return Bundle{} }

func (b *Bundle) Appname(string) *Bundle                                  { return nil }
func (b *Bundle) AuthMechanism(string) *Bundle                            { return nil }
func (b *Bundle) AuthMechaismProperties(map[string]string) *Bundle        { return nil }
func (b *Bundle) AuthSource(s string) *Bundle                             { return nil }
func (b *Bundle) ConnectTimeout(d time.Duration) *Bundle                  { return nil }
func (b *Bundle) Dialer(d ContextDialer) *Bundle                          { return nil }
func (b *Bundle) HeartbeatInterval(d time.Duration) *Bundle               { return nil }
func (b *Bundle) Hosts(s []string) *Bundle                                { return nil }
func (b *Bundle) Journal(v bool) *Bundle                                  { return nil }
func (b *Bundle) LocalThreshold(d time.Duration) *Bundle                  { return nil }
func (b *Bundle) MaxConnIdleTime(d time.Duration) *Bundle                 { return nil }
func (b *Bundle) MaxConnsPerHost(u uint16) *Bundle                        { return nil }
func (b *Bundle) MaxIdleConnsPerHost(u uint16) *Bundle                    { return nil }
func (b *Bundle) Password(s string) *Bundle                               { return nil }
func (b *Bundle) ReadConcernLevel(rc *readconcern.ReadConcern) *Bundle    { return nil }
func (b *Bundle) ReadPreference(s string) *Bundle                         { return nil }
func (b *Bundle) ReadPreferenceTagSets(m []map[string]string) *Bundle     { return nil }
func (b *Bundle) MaxStaleness(d time.Duration) *Bundle                    { return nil }
func (b *Bundle) ReplicaSet(s string) *Bundle                             { return nil }
func (b *Bundle) ServerSelectionTimeout(d time.Duration) *Bundle          { return nil }
func (b *Bundle) Single(v bool) *Bundle                                   { return nil }
func (b *Bundle) SocketTimeout(d time.Duration) *Bundle                   { return nil }
func (b *Bundle) SSL(v bool) *Bundle                                      { return nil }
func (b *Bundle) SSLClientCertificateKeyFile(s string) *Bundle            { return nil }
func (b *Bundle) SSLClientCertificateKeyPassword(s func() string) *Bundle { return nil }
func (b *Bundle) SSLInsecure(v bool) *Bundle                              { return nil }
func (b *Bundle) SSLCaFile(s string) *Bundle                              { return nil }
func (b *Bundle) WString(s string) *Bundle                                { return nil }
func (b *Bundle) WNumber(i int) *Bundle                                   { return nil }
func (b *Bundle) Username(s string) *Bundle                               { return nil }
func (b *Bundle) WTimeout(d time.Duration) *Bundle                        { return nil }
func (b *Bundle) URI(uri string) *Bundle                                  { return nil }

func (b *Bundle) String() string    { return "" }
func (b *Bundle) Unbundle() *Client { return nil }

func Appname(string) Option                                  { return nil }
func AuthMechanism(string) Option                            { return nil }
func AuthMechaismProperties(map[string]string) Option        { return nil }
func AuthSource(s string) Option                             { return nil }
func ConnectTimeout(d time.Duration) Option                  { return nil }
func Dialer(d ContextDialer) Option                          { return nil }
func HeartbeatInterval(d time.Duration) Option               { return nil }
func Hosts(s []string) Option                                { return nil }
func Journal(b bool) Option                                  { return nil }
func LocalThreshold(d time.Duration) Option                  { return nil }
func MaxConnIdleTime(d time.Duration) Option                 { return nil }
func MaxConnsPerHost(u uint16) Option                        { return nil }
func MaxIdleConnsPerHost(u uint16) Option                    { return nil }
func Password(s string) Option                               { return nil }
func ReadConcernLevel(rc *readconcern.ReadConcern) Option    { return nil }
func ReadPreference(s string) Option                         { return nil }
func ReadPreferenceTagSets(m []map[string]string) Option     { return nil }
func MaxStaleness(d time.Duration) Option                    { return nil }
func ReplicaSet(s string) Option                             { return nil }
func ServerSelectionTimeout(d time.Duration) Option          { return nil }
func Single(b bool) Option                                   { return nil }
func SocketTimeout(d time.Duration) Option                   { return nil }
func SSL(b bool) Option                                      { return nil }
func SSLOptionCertificateKeyFile(s string) Option            { return nil }
func SSLOptionCertificateKeyPassword(s func() string) Option { return nil }
func SSLInsecure(b bool) Option                              { return nil }
func SSLCaFile(s string) Option                              { return nil }
func WString(s string) Option                                { return nil }
func WNumber(i int) Option                                   { return nil }
func Username(s string) Option                               { return nil }
func WTimeout(d time.Duration) Option                        { return nil }
func URI(uri string) Option                                  { return nil }
