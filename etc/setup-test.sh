#!/usr/bin/env bash
#
# Set up test environment and write .test.env file.
set -eu

OS=${OS:-""}
SSL=${SSL:-nossl}
GO_BUILD_TAGS=${GO_BUILD_TAGS:-}
RACE=${RACE:-}
SERVERLESS=${SERVERLESS:-}
LOAD_BALANCER=${LOAD_BALANCER:-}
MONGODB_URI=${MONGODB_URI:-}
MONGOPROXY=${MONGOPROXY:-}
MONGO_PROX_URI=${MONGO_PROX_URI:-}

# Handle special cases first.
if [ -n "${TEST_ENTERPRISE_AUTH:-}" ]; then
  . $DRIVERS_TOOLS/.evergreen/secrets_handling/setup-secrets.sh drivers/enterprise_auth
  AUTH="auth"
  case $TEST_ENTERPRISE_AUTH in
  plain)
    MONGODB_URI="mongodb://${SASL_USER}:${SASL_PASS}@${SASL_HOST}:${SASL_PORT}/ldap?authMechanism=PLAIN"
    ;;
  gssapi)
    if [ "Windows_NT" = "${OS:-}" ]; then
      MONGODB_URI="mongodb://${PRINCIPAL/@/%40}:${SASL_PASS}@${SASL_HOST}:${SASL_PORT}/kerberos?authMechanism=GSSAPI"
    else
      echo ${KEYTAB_BASE64} | base64 -d >${PROJECT_DIRECTORY}/.evergreen/drivers.keytab
      mkdir -p ~/.krb5
      cat .evergreen/krb5.config | tee -a ~/.krb5/config
      kinit -k -t .evergreen/drivers.keytab -p "${PRINCIPAL}"
      MONGODB_URI="mongodb://${PRINCIPAL/@/%40}@${SASL_HOST}:${SASL_PORT}/kerberos?authMechanism=GSSAPI"
    fi
    ;;
  esac
  rm secrets-export.sh
fi

if [ -n "${SERVERLESS}" ]; then
  . $DRIVERS_TOOLS/.evergreen/serverless/secrets-export.sh
  MONGODB_URI="${SERVERLESS_URI}"
  AUTH="auth"
fi

if [ -n "${TEST_ATLAS_CONNECT:-}" ]; then
  . $DRIVERS_TOOLS/.evergreen/secrets_handling/setup-secrets.sh drivers/atlas_connect
fi

if [ -n "${LOAD_BALANCER}" ]; then
  # Verify that the required LB URI expansions are set to ensure that the test runner can correctly connect to
  # the LBs.
  if [ -z "${SINGLE_MONGOS_LB_URI}" ]; then
    echo "SINGLE_MONGOS_LB_URI must be set for testing against LBs"
    exit 1
  fi
  if [ -z "${MULTI_MONGOS_LB_URI}" ]; then
    echo "MULTI_MONGOS_LB_URI must be set for testing against LBs"
    exit 1
  fi
  MONGODB_URI="${SINGLE_MONGOS_LB_URI}"
fi

if [ -n "${MONGOPROXY}" ]; then
  echo "MONGOPROXY is set, using the proxy for qualifying tests."
  # If MONGOPROXY is set, we assume that the user wants to use the proxy for all tests.
  # This is useful for testing the proxy itself.
  MONGO_PROXY_URI="mongodb://127.0.0.1:28017/?directConnection=true"
fi

if [ -n "${OCSP_ALGORITHM:-}" ]; then
  MONGO_GO_DRIVER_CA_FILE="${DRIVERS_TOOLS}/.evergreen/ocsp/${OCSP_ALGORITHM}/ca.pem"
  if [ "Windows_NT" = "$OS" ]; then
    MONGO_GO_DRIVER_CA_FILE=$(cygpath -m $MONGO_GO_DRIVER_CA_FILE)
  fi
fi

# Handle encryption.
if [[ "${GO_BUILD_TAGS}" =~ cse ]]; then
  # Install libmongocrypt if needed.
  task install-libmongocrypt

  # Handle libmongocrypt paths.
  PKG_CONFIG_PATH=$(pwd)/install/libmongocrypt/lib64/pkgconfig
  LD_LIBRARY_PATH=$(pwd)/install/libmongocrypt/lib64

  if [ "$(uname -s)" = "Darwin" ]; then
    PKG_CONFIG_PATH=$(pwd)/install/libmongocrypt/lib/pkgconfig
    DYLD_FALLBACK_LIBRARY_PATH=$(pwd)/install/libmongocrypt/lib
  fi

  if [ "${SKIP_CRYPT_SHARED_LIB:-''}" = "true" ]; then
    CRYPT_SHARED_LIB_PATH=""
    echo "crypt_shared library is skipped"
  elif [ -z "${CRYPT_SHARED_LIB_PATH:-}" ]; then
    echo "crypt_shared library path is empty"
  else
    echo "crypt_shared library will be loaded from path: $CRYPT_SHARED_LIB_PATH"
  fi
fi

# Handle the build tags argument.
if [ -n "${GO_BUILD_TAGS}" ]; then
  BUILD_TAGS="${RACE} --tags=${GO_BUILD_TAGS}"
else
  BUILD_TAGS="${RACE}"
fi

# Handle certificates.
if [ "$SSL" != "nossl" ] && [ -z "${SERVERLESS}" ] && [ -z "${OCSP_ALGORITHM:-}" ]; then
  MONGO_GO_DRIVER_CA_FILE="$DRIVERS_TOOLS/.evergreen/x509gen/ca.pem"
  MONGO_GO_DRIVER_KEY_FILE="$DRIVERS_TOOLS/.evergreen/x509gen/client.pem"
  MONGO_GO_DRIVER_PKCS8_ENCRYPTED_KEY_FILE="$DRIVERS_TOOLS/.evergreen/x509gen/client-pkcs8-encrypted.pem"
  MONGO_GO_DRIVER_PKCS8_UNENCRYPTED_KEY_FILE="$DRIVERS_TOOLS/.evergreen/x509gen/client-pkcs8-unencrypted.pem"

  if [ "Windows_NT" = "$OS" ]; then
    MONGO_GO_DRIVER_CA_FILE=$(cygpath -m $MONGO_GO_DRIVER_CA_FILE)
    MONGO_GO_DRIVER_KEY_FILE=$(cygpath -m $MONGO_GO_DRIVER_KEY_FILE)
    MONGO_GO_DRIVER_PKCS8_ENCRYPTED_KEY_FILE=$(cygpath -m $MONGO_GO_DRIVER_PKCS8_ENCRYPTED_KEY_FILE)
    MONGO_GO_DRIVER_PKCS8_UNENCRYPTED_KEY_FILE=$(cygpath -m $MONGO_GO_DRIVER_PKCS8_UNENCRYPTED_KEY_FILE)
  fi
fi

cat <<EOT >.test.env
AUTH="${AUTH:-}"
SSL="${SSL}"
MONGO_GO_DRIVER_CA_FILE="${MONGO_GO_DRIVER_CA_FILE:-}"
MONGO_GO_DRIVER_KEY_FILE="${MONGO_GO_DRIVER_KEY_FILE:-}"
MONGO_GO_DRIVER_PKCS8_ENCRYPTED_KEY_FILE="${MONGO_GO_DRIVER_PKCS8_ENCRYPTED_KEY_FILE:-}"
MONGO_GO_DRIVER_PKCS8_UNENCRYPTED_KEY_FILE="${MONGO_GO_DRIVER_PKCS8_UNENCRYPTED_KEY_FILE:-}"
TOPOLOGY="${TOPOLOGY:-}"
SERVERLESS="${SERVERLESS}"
REQUIRE_API_VERSION="${REQUIRE_API_VERSION:-}"
LOAD_BALANCER="${LOAD_BALANCER}"
MONGO_GO_DRIVER_COMPRESSOR="${MONGO_GO_DRIVER_COMPRESSOR:-}"
BUILD_TAGS="${BUILD_TAGS}"
CRYPT_SHARED_LIB_PATH="${CRYPT_SHARED_LIB_PATH:-}"
PKG_CONFIG_PATH="${PKG_CONFIG_PATH:-}"
LD_LIBRARY_PATH="${LD_LIBRARY_PATH:-}"
MACOS_LIBRARY_PATH="${DYLD_FALLBACK_LIBRARY_PATH:-}"
SKIP_CSOT_TESTS=${SKIP_CSOT_TESTS:-}
MONGO_PROXY_URI="${MONGO_PROXY_URI:-}"
EOT

if [ -n "${MONGODB_URI}" ]; then
  echo "MONGODB_URI=\"${MONGODB_URI}\"" >>.test.env
fi

if [ -n "${SERVERLESS}" ]; then
  echo "SERVERLESS_ATLAS_USER=$SERVERLESS_ATLAS_USER" >>.test.env
  echo "SERVERLESS_ATLAS_PASSWORD=$SERVERLESS_ATLAS_PASSWORD" >>.test.env
fi

if [ -n "${LOAD_BALANCER}" ]; then
  echo "SINGLE_MONGOS_LB_URI=${SINGLE_MONGOS_LB_URI}" >>.test.env
  echo "MULTI_MONGOS_LB_URI=${MULTI_MONGOS_LB_URI}" >>.test.env
fi

# Add secrets to the test file.
if [ -f "secrets-export.sh" ]; then
  while read p; do
    if [[ "$p" =~ ^export ]]; then
      echo "$p" | sed 's/export //' >>.test.env
    fi
  done <secrets-export.sh
fi
