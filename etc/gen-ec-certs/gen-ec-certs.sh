# This script is used to generate Elliptic Curve (EC) certificates.
# The EC certificates are used for testing the Go driver with PyKMIP.
# PyKMIP does not support Go's default TLS cipher suites with RSA.
# See: GODRIVER-2239. 
set -euo pipefail
CA_SERIAL=$RANDOM
SERVER_SERIAL=$RANDOM
CLIENT_SERIAL=$RANDOM
DAYS=14600

# Generate CA certificate ... begin
# Generate an EC private key.
openssl ecparam -name prime256v1 -genkey -out ca-ec.key -noout
# Generate a certificate signing request.
openssl req -new -key ca-ec.key -out ca-ec.csr -subj "/C=US/ST=New York/L=New York City/O=MongoDB/OU=DBX/CN=ca/" -config empty.cnf -sha256
# Self-sign the request.
openssl x509 -in ca-ec.csr -out ca-ec.pem -req -signkey ca-ec.key -days $DAYS -sha256 -set_serial $CA_SERIAL
# Generate CA certificate ... end

# Generate Server certificate ... begin
# Generate an EC private key.
openssl ecparam -name prime256v1 -genkey -out server-ec.key -noout
# Generate a certificate signing request.
openssl req -new -key server-ec.key -out server-ec.csr -subj "/C=US/ST=New York/L=New York City/O=MongoDB/OU=DBX/CN=server/" -config empty.cnf -sha256
# Sign the request with the CA. Add server extensions.
openssl x509 -in server-ec.csr -out server-ec.pem -req -CA ca-ec.pem -CAkey ca-ec.key -days $DAYS -sha256 -set_serial $SERVER_SERIAL -extfile server.ext
# Append private key to .pem file.
cat server-ec.key >> server-ec.pem
# Generate Server certificate ... end

# Generate Client certificate ... begin
# Generate an EC private key.
openssl ecparam -name prime256v1 -genkey -out client-ec.key -noout
# Generate a certificate signing request.
# Use the Common Name (CN) of "client". PyKMIP identifies the client by the CN. The test server expects the identity of "client".
openssl req -new -key client-ec.key -out client-ec.csr -subj "/C=US/ST=New York/L=New York City/O=MongoDB/OU=DBX/CN=client/" -config empty.cnf -sha256
# Sign the request with the CA. Add client extensions.
openssl x509 -in client-ec.csr -out client-ec.pem -req -CA ca-ec.pem -CAkey ca-ec.key -days $DAYS -sha256 -set_serial $CLIENT_SERIAL -extfile client.ext
# Append private key to .pem file.
cat client-ec.key >> client-ec.pem
# Generate Client certificate ... end

# Clean-up.
rm *.csr
rm *.key
