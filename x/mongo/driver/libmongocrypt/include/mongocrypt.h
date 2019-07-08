/*
 * Copyright 2019-present MongoDB, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
#ifndef MONGOCRYPT_H
#define MONGOCRYPT_H

/** @file mongocrypt.h The top-level handle to libmongocrypt. */

/**
 * @mainpage libmongocrypt
 * See all public API documentation in: @ref mongocrypt.h
 */

#include "mongocrypt-export.h"
#include "mongocrypt-compat.h"

/**
 * @def MONGOCRYPT_VERSION
 * The version string x.y.z or x.y.z-pre for libmongocrypt
 */
#define MONGOCRYPT_VERSION "1.0.0-beta2"

/**
 * Returns the version string for libmongocrypt.
 *
 * @param[out] len  An optional length of the returned string. May be NULL.
 * @returns a NULL terminated version string for libmongocrypt.
 */
MONGOCRYPT_EXPORT
const char *
mongocrypt_version (uint32_t *len);


/**
 * A non-owning view of a byte buffer.
 *
 * When constructing a mongocrypt_binary_t it is the responsibility of the
 * caller to maintain the lifetime of the viewed data. However, all public
 * functions that take a mongocrypt_binary_t as an argument will make a copy of
 * the viewed data. For example, the following is valid:
 *
 * @code{.c}
 * mongocrypt_binary_t bin = mongocrypt_binary_new_from_data(mydata, mylen);
 * assert (mongocrypt_setopt_kms_provider_local (crypt), bin);
 * // The viewed data of bin has been copied. Ok to free the view and the data.
 * mongocrypt_binary_destroy (bin);
 * my_free_fn (mydata);
 * @endcode
 *
 * Functions with a mongocrypt_binary_t* out guarantee the lifetime of the
 * viewed data to live as long as the parent object. For example, @ref
 * mongocrypt_ctx_mongo_op guarantees that the viewed data of
 * mongocrypt_binary_t is valid until the parent ctx is destroyed with @ref
 * mongocrypt_ctx_destroy.
 */
typedef struct _mongocrypt_binary_t mongocrypt_binary_t;


/**
 * Create a new non-owning view of a buffer (data + length).
 *
 * Use this to create a mongocrypt_binary_t used for output parameters.
 *
 * @returns A new mongocrypt_binary_t.
 */
MONGOCRYPT_EXPORT
mongocrypt_binary_t *
mongocrypt_binary_new (void);


/**
 * Create a new non-owning view of a buffer (data + length).
 *
 * @param[in] data A pointer to an array of bytes. This data is not copied. @p
 * data must outlive the binary object.
 * @param[in] len The length of the @p data byte array.
 *
 * @returns A new @ref mongocrypt_binary_t.
 */
MONGOCRYPT_EXPORT
mongocrypt_binary_t *
mongocrypt_binary_new_from_data (uint8_t *data, uint32_t len);


/**
 * Get a pointer to the viewed data.
 *
 * @param[in] binary The @ref mongocrypt_binary_t.
 *
 * @returns A pointer to the viewed data.
 */
MONGOCRYPT_EXPORT
const uint8_t *
mongocrypt_binary_data (const mongocrypt_binary_t *binary);


/**
 * Get the length of the viewed data.
 *
 * @param[in] binary The @ref mongocrypt_binary_t.
 *
 * @returns The length of the viewed data.
 */
MONGOCRYPT_EXPORT
uint32_t
mongocrypt_binary_len (const mongocrypt_binary_t *binary);


/**
 * Free the @ref mongocrypt_binary_t.
 *
 * This does not free the viewed data.
 *
 * @param[in] binary The mongocrypt_binary_t destroy.
 */
MONGOCRYPT_EXPORT
void
mongocrypt_binary_destroy (mongocrypt_binary_t *binary);


/**
 * Indicates success or contains error information.
 *
 * Functions like @ref mongocrypt_ctx_encrypt_init follow a pattern to expose a
 * status. A boolean is returned. True indicates success, and false indicates
 * failure. On failure a status on the handle is set, and is accessible with a
 * corresponding (handle)_status function. E.g. @ref mongocrypt_ctx_status.
 */
typedef struct _mongocrypt_status_t mongocrypt_status_t;

/**
 * Indicates the type of error.
 */
typedef enum {
   MONGOCRYPT_STATUS_OK = 0,
   MONGOCRYPT_STATUS_ERROR_CLIENT = 1,
   MONGOCRYPT_STATUS_ERROR_KMS = 2
} mongocrypt_status_type_t;


/**
 * Create a new status object.
 *
 * Use a new status object to retrieve the status from a handle by passing
 * this as an out-parameter to functions like @ref mongocrypt_ctx_status.
 * When done, destroy it with @ref mongocrypt_status_destroy.
 *
 * @returns A new status object.
 */
MONGOCRYPT_EXPORT
mongocrypt_status_t *
mongocrypt_status_new (void);


/**
 * Indicates success or the type of error.
 *
 * @param[in] status The status object.
 *
 * @returns A @ref mongocrypt_status_type_t.
 */
MONGOCRYPT_EXPORT
mongocrypt_status_type_t
mongocrypt_status_type (mongocrypt_status_t *status);


/**
 * Get an error code or 0.
 *
 * @param[in] status The status object.
 *
 * @returns An error code.
 */
MONGOCRYPT_EXPORT
uint32_t
mongocrypt_status_code (mongocrypt_status_t *status);


/**
 * Get the error message associated with a status or NULL.
 *
 * @param[in] status The status object.
 * @param[out] len An optional length of the returned string. May be NULL.
 *
 * @returns A NULL terminated error message or NULL.
 */
MONGOCRYPT_EXPORT
const char *
mongocrypt_status_message (mongocrypt_status_t *status, uint32_t *len);


/**
 * Returns true if the status indicates success.
 *
 * @param[in] status The status to check.
 *
 * @returns A boolean indicating success. If false, an error status is set.
 * Retrieve it with @ref mongocrypt_ctx_status
 */
MONGOCRYPT_EXPORT
bool
mongocrypt_status_ok (mongocrypt_status_t *status);


/**
 * Free the memory for a status object.
 *
 * @param[in] status The status to destroy.
 */
MONGOCRYPT_EXPORT
void
mongocrypt_status_destroy (mongocrypt_status_t *status);

/**
 * Indicates the type of log message.
 */
typedef enum {
   MONGOCRYPT_LOG_LEVEL_FATAL = 0,
   MONGOCRYPT_LOG_LEVEL_ERROR = 1,
   MONGOCRYPT_LOG_LEVEL_WARNING = 2,
   MONGOCRYPT_LOG_LEVEL_INFO = 3,
   MONGOCRYPT_LOG_LEVEL_TRACE = 4
} mongocrypt_log_level_t;


/**
 * A log callback function. Set a custom log callback with @ref
 * mongocrypt_setopt_log_handler.
 *
 * @param[in] message A NULL terminated message.
 * @param[in] message_len The length of message.
 * @param[in] ctx A context provided by the caller of @ref
 * mongocrypt_setopt_log_handler.
 */
typedef void (*mongocrypt_log_fn_t) (mongocrypt_log_level_t level,
                                     const char *message,
                                     uint32_t message_len,
                                     void *ctx);


/**
 * The top-level handle to libmongocrypt.
 *
 * Create a mongocrypt_t handle to perform operations within libmongocrypt:
 * encryption, decryption, registering log callbacks, etc.
 *
 * Functions on a mongocrypt_t are thread safe, though functions on derived
 * handles (e.g. mongocrypt_ctx_t) are not and must be owned by a single
 * thread. See each handle's documentation for thread-safety considerations.
 *
 * Multiple mongocrypt_t handles may be created.
 */
typedef struct _mongocrypt_t mongocrypt_t;


/**
 * Allocate a new @ref mongocrypt_t object.
 *
 * Set options using mongocrypt_setopt_* functions, then initialize with @ref
 * mongocrypt_init. When done with the @ref mongocrypt_t, free with @ref
 * mongocrypt_destroy.
 *
 * @returns A new @ref mongocrypt_t object.
 */
MONGOCRYPT_EXPORT
mongocrypt_t *
mongocrypt_new (void);


/**
 * Set a handler on the @ref mongocrypt_t object to get called on every log
 * message.
 *
 * @param[in] crypt The @ref mongocrypt_t object.
 * @param[in] log_fn The log callback.
 * @param[in] log_ctx A context passed as an argument to the log callback every
 * invokation.
 * @pre @ref mongocrypt_init has not been called on @p crypt.
 * @returns A boolean indicating success. If false, an error status is set.
 * Retrieve it with @ref mongocrypt_ctx_status
 */
MONGOCRYPT_EXPORT
bool
mongocrypt_setopt_log_handler (mongocrypt_t *crypt,
                               mongocrypt_log_fn_t log_fn,
                               void *log_ctx);


/**
 * Configure an AWS KMS provider on the @ref mongocrypt_t object.
 *
 * @param[in] crypt The @ref mongocrypt_t object.
 * @param[in] aws_access_key_id The AWS access key ID used to generate KMS
 * messages.
 * @param[in] aws_access_key_id_len The string length (in bytes) of @p
 * aws_access_key_id. Pass -1 to determine the string length with strlen (must
 * be NULL terminated).
 * @param[in] aws_secret_access_key The AWS secret access key used to generate
 * KMS messages.
 * @param[in] aws_secret_access_key_len The string length (in bytes) of @p
 * aws_secret_access_key. Pass -1 to determine the string length with strlen
 * (must be NULL terminated).
 * @pre @ref mongocrypt_init has not been called on @p crypt.
 * @returns A boolean indicating success. If false, an error status is set.
 * Retrieve it with @ref mongocrypt_ctx_status
 */
MONGOCRYPT_EXPORT
bool
mongocrypt_setopt_kms_provider_aws (mongocrypt_t *crypt,
                                    const char *aws_access_key_id,
                                    int32_t aws_access_key_id_len,
                                    const char *aws_secret_access_key,
                                    int32_t aws_secret_access_key_len);


/**
 * Configure a local KMS provider on the @ref mongocrypt_t object.
 *
 * @param[in] crypt The @ref mongocrypt_t object.
 * @param[in] key A 64 byte master key used to encrypt and decrypt key vault
 * keys. The viewed data is copied. It is valid to destroy @p key with @ref
 * mongocrypt_binary_destroy immediately after.
 * @pre @ref mongocrypt_init has not been called on @p crypt.
 * @returns A boolean indicating success. If false, an error status is set.
 * Retrieve it with @ref mongocrypt_ctx_status
 */
MONGOCRYPT_EXPORT
bool
mongocrypt_setopt_kms_provider_local (mongocrypt_t *crypt,
                                      mongocrypt_binary_t *key);


/**
 * Set a local schema map for encryption.
 *
 * @param[in] crypt The @ref mongocrypt_t object.
 * @param[in] schema_map A BSON document representing the schema map supplied by
 * the user. The keys are collection namespaces and values are JSON schemas. The
 * viewed data copied. It is valid to destroy @p schema_map with @ref
 * mongocrypt_binary_destroy immediately after.
 * @pre @p crypt has not been initialized.
 * @returns A boolean indicating success. If false, an error status is set.
 * Retrieve it with @ref mongocrypt_status
 */
MONGOCRYPT_EXPORT
bool
mongocrypt_setopt_schema_map (mongocrypt_t *crypt,
                              mongocrypt_binary_t *schema_map);


/**
 * Initialize new @ref mongocrypt_t object.
 *
 * Set options before using @ref mongocrypt_setopt_kms_provider_local, @ref
 * mongocrypt_setopt_kms_provider_aws, or @ref mongocrypt_setopt_log_handler.
 *
 * @param[in] crypt The @ref mongocrypt_t object.
 *
 * @returns A boolean indicating success. If false, an error status is set.
 * Retrieve it with @ref mongocrypt_ctx_status Failure may occur if previously
 * set
 * options are invalid.
 */
MONGOCRYPT_EXPORT
bool
mongocrypt_init (mongocrypt_t *crypt);


/**
 * Get the status associated with a @ref mongocrypt_t object.
 *
 * @param[in] crypt The @ref mongocrypt_t object.
 * @param[out] status Receives the status.
 *
 * @returns A boolean indicating success. If false, an error status is set.
 * Retrieve it with @ref mongocrypt_ctx_status
 */
MONGOCRYPT_EXPORT
bool
mongocrypt_status (mongocrypt_t *crypt, mongocrypt_status_t *status);


/**
 * Destroy the @ref mongocrypt_t object.
 *
 * @param[in] crypt The @ref mongocrypt_t object to destroy.
 */
MONGOCRYPT_EXPORT
void
mongocrypt_destroy (mongocrypt_t *crypt);


/**
 * Manages the state machine for encryption or decryption.
 */
typedef struct _mongocrypt_ctx_t mongocrypt_ctx_t;


/**
 * Create a new uninitialized @ref mongocrypt_ctx_t.
 *
 * Initialize the context with functions like @ref mongocrypt_ctx_encrypt_init.
 * When done, destroy it with @ref mongocrypt_ctx_destroy.
 *
 * @param[in] crypt The @ref mongocrypt_t object.
 * @returns A new context.
 */
MONGOCRYPT_EXPORT
mongocrypt_ctx_t *
mongocrypt_ctx_new (mongocrypt_t *crypt);


/**
 * Get the status associated with a @ref mongocrypt_ctx_t object.
 *
 * @param[in] ctx The @ref mongocrypt_ctx_t object.
 * @param[out] status Receives the status.
 *
 * @returns True if the output is an ok status, false if it is an error
 * status.
 *
 * @see mongocrypt_status_ok
 */
MONGOCRYPT_EXPORT
bool
mongocrypt_ctx_status (mongocrypt_ctx_t *ctx, mongocrypt_status_t *status);


/**
 * Set the key id to use for explicit encryption.
 *
 * It is an error to set both this and the key alt name.
 *
 * @param[in] ctx The @ref mongocrypt_ctx_t object.
 * @param[in] key_id The key_id to use. The viewed data is copied. It is valid
 * to destroy @p key_id with @ref mongocrypt_binary_destroy immediately after.
 * @pre @p ctx has not been initialized.
 * @returns A boolean indicating success. If false, an error status is set.
 * Retrieve it with @ref mongocrypt_ctx_status
 */
MONGOCRYPT_EXPORT
bool
mongocrypt_ctx_setopt_key_id (mongocrypt_ctx_t *ctx,
                              mongocrypt_binary_t *key_id);

/**
 * Set the keyAltName to use for explicit encryption.
 * keyAltName should be a binary encoding a bson document
 * with the following format:
 *
 *   { "keyAltName" : (BSON UTF8 value) }
 *
 * It is an error to set both this and the key id.
 *
 * @param[in] ctx The @ref mongocrypt_ctx_t object.
 * @param[in] key_alt_name The name to use. The viewed data is copied. It is
 * valid to destroy @p key_alt_name with @ref mongocrypt_binary_destroy
 * immediately after.
 * @pre @p ctx has not been initialized.
 * @returns A boolean indicating success. If false, an error status is set.
 * Retrieve it with @ref mongocrypt_ctx_status
 */
MONGOCRYPT_EXPORT
bool
mongocrypt_ctx_setopt_key_alt_name (mongocrypt_ctx_t *ctx,
                                    mongocrypt_binary_t *key_alt_name);

/**
 * Set the algorithm used for encryption to either
 * deterministic or random encryption. This value
 * should only be set when using explicit encryption.
 *
 * If -1 is passed in for "len", then "algorithm" is
 * assumed to be a null-terminated string.
 *
 * Valid values for algorithm are:
 *   "AEAD_AES_256_CBC_HMAC_SHA_512-Deterministic"
 *   "AEAD_AES_256_CBC_HMAC_SHA_512-Randomized"
 *
 * @param[in] ctx The @ref mongocrypt_ctx_t object.
 * @param[in] algorithm A string specifying the algorithm to
 * use for encryption.
 * @param[in] len The length of the algorithm string.
 * @pre @p ctx has not been initialized.
 * @returns A boolean indicating success. If false, an error status is set.
 * Retrieve it with @ref mongocrypt_ctx_status
 */
MONGOCRYPT_EXPORT
bool
mongocrypt_ctx_setopt_algorithm (mongocrypt_ctx_t *ctx,
                                 const char *algorithm,
                                 int len);


/**
 * Identify the AWS KMS master key to use for creating a data key.
 *
 * @param[in] ctx The @ref mongocrypt_ctx_t object.
 * @param[in] region The AWS region.
 * @param[in] region_len The string length of @p region. Pass -1 to determine
 * the string length with strlen (must be NULL terminated).
 * @param[in] cmk The Amazon Resource Name (ARN) of the customer master key
 * (CMK).
 * @param[in] cmk_len The string length of @p cmk_len. Pass -1 to determine the
 * string length with strlen (must be NULL terminated).
 * @pre @p ctx has not been initialized.
 * @returns A boolean indicating success. If false, an error status is set.
 * Retrieve it with @ref mongocrypt_ctx_status
 */
MONGOCRYPT_EXPORT
bool
mongocrypt_ctx_setopt_masterkey_aws (mongocrypt_ctx_t *ctx,
                                     const char *region,
                                     int32_t region_len,
                                     const char *cmk,
                                     int32_t cmk_len);


/**
 * Set the master key to "local" for creating a data key.
 *
 * @param[in] ctx The @ref mongocrypt_ctx_t object.
 * @pre @p ctx has not been initialized.
 * @returns A boolean indicating success. If false, an error status is set.
 * Retrieve it with @ref mongocrypt_ctx_status
 */
MONGOCRYPT_EXPORT
bool
mongocrypt_ctx_setopt_masterkey_local (mongocrypt_ctx_t *ctx);


/**
 * Initialize a context to create a data key.
 *
 * Associated options:
 * - @ref mongocrypt_ctx_setopt_masterkey_aws
 * - @ref mongocrypt_ctx_setopt_masterkey_local
 *
 * @param[in] ctx The @ref mongocrypt_ctx_t object.
 * @returns A boolean indicating success. If false, an error status is set.
 * Retrieve it with @ref mongocrypt_ctx_status
 * @pre A master key option has been set, and an associated KMS provider
 * has been set on the parent @ref mongocrypt_t.
 */
MONGOCRYPT_EXPORT
bool
mongocrypt_ctx_datakey_init (mongocrypt_ctx_t *ctx);

/**
 * Initialize a context for encryption.
 *
 * @param[in] ctx The @ref mongocrypt_ctx_t object.
 * @param[in] db The database name.
 * @param[in] db_len The byte length of @p db. Pass -1 to determine the string
 * length with strlen (must
 * be NULL terminated).
 * @param[in] cmd The BSON command to be encrypted. The viewed data is copied.
 * It is valid to destroy @p cmd with @ref mongocrypt_binary_destroy immediately
 * after.
 * @returns A boolean indicating success. If false, an error status is set.
 * Retrieve it with @ref mongocrypt_ctx_status
 */
MONGOCRYPT_EXPORT
bool
mongocrypt_ctx_encrypt_init (mongocrypt_ctx_t *ctx,
                             const char *db,
                             int32_t db_len,
                             mongocrypt_binary_t *cmd);

/**
 * Explicit helper method to encrypt a single BSON object. Contexts
 * created for explicit encryption will not go through mongocryptd.
 *
 * To specify a key_id, algorithm, or iv to use, please use the
 * corresponding mongocrypt_setopt methods before calling this.
 *
 * This method expects the passed-in BSON to be of the form:
 * { "v" : BSON value to encrypt }
 *
 * Associated options:
 * - @ref mongocrypt_ctx_setopt_key_id
 * - @ref mongocrypt_ctx_setopt_key_alt_name
 * - @ref mongocrypt_ctx_setopt_algorithm
 *
 * @param[in] ctx A @ref mongocrypt_ctx_t.
 * @param[in] msg A @ref mongocrypt_binary_t the plaintext BSON value. The
 * viewed data is copied. It is valid to destroy @p msg with @ref
 * mongocrypt_binary_destroy immediately after.
 * @returns A boolean indicating success. If false, an error status is set.
 * Retrieve it with @ref mongocrypt_ctx_status
 */
MONGOCRYPT_EXPORT
bool
mongocrypt_ctx_explicit_encrypt_init (mongocrypt_ctx_t *ctx,
                                      mongocrypt_binary_t *msg);


/**
 * Initialize a context for decryption.
 *
 * @param[in] ctx The @ref mongocrypt_ctx_t object.
 * @param[in] doc The document to be decrypted. The viewed data is copied. It is
 * valid to destroy @p doc with @ref mongocrypt_binary_destroy immediately
 * after.
 * @returns A boolean indicating success. If false, an error status is set.
 * Retrieve it with @ref mongocrypt_ctx_status
 */
MONGOCRYPT_EXPORT
bool
mongocrypt_ctx_decrypt_init (mongocrypt_ctx_t *ctx, mongocrypt_binary_t *doc);


/**
 * Explicit helper method to decrypt a single BSON object.
 *
 *
 * @param[in] ctx A @ref mongocrypt_ctx_t.
 * @param[in] msg A @ref mongocrypt_binary_t the encrypted BSON. The viewed data
 * is copied. It is valid to destroy @p msg with @ref mongocrypt_binary_destroy
 * immediately after.
 */
MONGOCRYPT_EXPORT
bool
mongocrypt_ctx_explicit_decrypt_init (mongocrypt_ctx_t *ctx,
                                      mongocrypt_binary_t *msg);


/**
 * Indicates the state of the @ref mongocrypt_ctx_t. Each state requires
 * different handling. See [the integration guide](https://github.com/mongodb/libmongocrypt/blob/master/integrating.md#state-machine)
 * for information on what to do for each state.
 */
typedef enum {
   MONGOCRYPT_CTX_ERROR = 0,
   MONGOCRYPT_CTX_NEED_MONGO_COLLINFO = 1, /* run on main MongoClient */
   MONGOCRYPT_CTX_NEED_MONGO_MARKINGS = 2, /* run on mongocryptd. */
   MONGOCRYPT_CTX_NEED_MONGO_KEYS = 3,     /* run on key vault */
   MONGOCRYPT_CTX_NEED_KMS = 4,
   MONGOCRYPT_CTX_READY = 5, /* ready for encryption/decryption */
   MONGOCRYPT_CTX_DONE = 6
} mongocrypt_ctx_state_t;


/**
 * Get the current state of a context.
 *
 * @param[in] ctx The @ref mongocrypt_ctx_t object.
 * @returns A @ref mongocrypt_ctx_state_t.
 */
MONGOCRYPT_EXPORT
mongocrypt_ctx_state_t
mongocrypt_ctx_state (mongocrypt_ctx_t *ctx);


/**
 * Get BSON necessary to run the mongo operation when mongocrypt_ctx_t
 * is in MONGOCRYPT_CTX_NEED_MONGO_* states.
 *
 * @p op_bson is a BSON document to be used for the operation.
 * - For MONGOCRYPT_CTX_NEED_MONGO_COLLINFO it is a listCollections filter.
 * - For MONGOCRYPT_CTX_NEED_MONGO_KEYS it is a find filter.
 * - For MONGOCRYPT_CTX_NEED_MONGO_MARKINGS it is a command to send to
 * mongocryptd.
 *
 * The lifetime of @p op_bson is tied to the lifetime of @p ctx. It is valid
 * until @ref mongocrypt_ctx_destroy is called.
 *
 * @param[in] ctx The @ref mongocrypt_ctx_t object.
 * @param[out] op_bson A BSON document for the MongoDB operation. The data
 * viewed by @p op_bson is guaranteed to be valid until @p ctx is destroyed with
 * @ref mongocrypt_ctx_destroy.
 * @returns A boolean indicating success. If false, an error status is set.
 * Retrieve it with @ref mongocrypt_ctx_status
 */
MONGOCRYPT_EXPORT
bool
mongocrypt_ctx_mongo_op (mongocrypt_ctx_t *ctx, mongocrypt_binary_t *op_bson);


/**
 * Feed a BSON reply or result when when mongocrypt_ctx_t is in
 * MONGOCRYPT_CTX_NEED_MONGO_* states. This may be called multiple times
 * depending on the operation.
 *
 * op_bson is a BSON document to be used for the operation.
 * - For MONGOCRYPT_CTX_NEED_MONGO_COLLINFO it is a doc from a listCollections
 * cursor.
 * - For MONGOCRYPT_CTX_NEED_MONGO_KEYS it is a doc from a find cursor.
 * - For MONGOCRYPT_CTX_NEED_MONGO_MARKINGS it is a reply from mongocryptd.
 *
 * @param[in] ctx The @ref mongocrypt_ctx_t object.
 * @param[in] reply A BSON document for the MongoDB operation. The viewed data
 * is copied. It is valid to destroy @p reply with @ref
 * mongocrypt_binary_destroy immediately after.
 * @returns A boolean indicating success. If false, an error status is set.
 * Retrieve it with @ref mongocrypt_ctx_status
 */
MONGOCRYPT_EXPORT
bool
mongocrypt_ctx_mongo_feed (mongocrypt_ctx_t *ctx, mongocrypt_binary_t *reply);


/**
 * Call when done feeding the reply (or replies) back to the context.
 *
 * @param[in] ctx The @ref mongocrypt_ctx_t object.
 * @returns A boolean indicating success. If false, an error status is set.
 * Retrieve it with @ref mongocrypt_ctx_status
 */
MONGOCRYPT_EXPORT
bool
mongocrypt_ctx_mongo_done (mongocrypt_ctx_t *ctx);


/**
 * Manages a single KMS HTTP request/response.
 */
typedef struct _mongocrypt_kms_ctx_t mongocrypt_kms_ctx_t;


/**
 * Get the next KMS handle.
 *
 * Multiple KMS handles may be retrieved at once. Drivers may do this to fan
 * out multiple concurrent KMS HTTP requests. Feeding multiple KMS requests
 * is thread-safe.
 *
 * If KMS handles are being handled synchronously, the driver can reuse the same
 * TLS socket to send HTTP requests and receive responses.
 *
 * @param[in] ctx A @ref mongocrypt_ctx_t.
 * @returns a new @ref mongocrypt_kms_ctx_t or NULL.
 */
MONGOCRYPT_EXPORT
mongocrypt_kms_ctx_t *
mongocrypt_ctx_next_kms_ctx (mongocrypt_ctx_t *ctx);


/**
 * Get the HTTP request message for a KMS handle.
 *
 * The lifetime of @p msg is tied to the lifetime of @p kms. It is valid
 * until @ref mongocrypt_ctx_kms_done is called.
 *
 * @param[in] kms A @ref mongocrypt_kms_ctx_t.
 * @param[out] msg The HTTP request to send to KMS. The data viewed by @p msg is
 * guaranteed to be valid until the call of @ref mongocrypt_ctx_kms_done of the
 * parent @ref mongocrypt_ctx_t.
 * @returns A boolean indicating success. If false, an error status is set.
 * Retrieve it with @ref mongocrypt_ctx_status
 */
MONGOCRYPT_EXPORT
bool
mongocrypt_kms_ctx_message (mongocrypt_kms_ctx_t *kms,
                            mongocrypt_binary_t *msg);


/**
 * Get the hostname from which to connect over TLS.
 *
 * The storage for @p endpoint is not owned by the caller, but
 * is valid until calling @ref mongocrypt_ctx_kms_done.
 *
 * @param[in] kms A @ref mongocrypt_kms_ctx_t.
 * @param[out] endpoint The output hostname as a NULL terminated string.
 * @returns A boolean indicating success. If false, an error status is set.
 * Retrieve it with @ref mongocrypt_ctx_status
 */
MONGOCRYPT_EXPORT
bool
mongocrypt_kms_ctx_endpoint (mongocrypt_kms_ctx_t *kms, const char **endpoint);


/**
 * Indicates how many bytes to feed into @ref mongocrypt_kms_ctx_feed.
 *
 * @param[in] kms The @ref mongocrypt_kms_ctx_t.
 * @returns The number of requested bytes.
 */
MONGOCRYPT_EXPORT
uint32_t
mongocrypt_kms_ctx_bytes_needed (mongocrypt_kms_ctx_t *kms);


/**
 * Feed bytes from the HTTP response.
 *
 * Feeding more bytes than what has been returned in @ref
 * mongocrypt_kms_ctx_bytes_needed is an error.
 *
 * @param[in] kms The @ref mongocrypt_kms_ctx_t.
 * @param[in] bytes The bytes to feed. The viewed data is copied. It is valid to
 * destroy @p bytes with @ref mongocrypt_binary_destroy immediately after.
 * @returns A boolean indicating success. If false, an error status is set.
 * Retrieve it with @ref mongocrypt_ctx_status
 */
MONGOCRYPT_EXPORT
bool
mongocrypt_kms_ctx_feed (mongocrypt_kms_ctx_t *kms, mongocrypt_binary_t *bytes);


/**
 * Get the status associated with a @ref mongocrypt_kms_ctx_t object.
 *
 * @param[in] kms The @ref mongocrypt_kms_ctx_t object.
 * @param[out] status Receives the status.
 *
 * @returns A boolean indicating success. If false, an error status is set.
 * Retrieve it with @ref mongocrypt_ctx_status
 */
MONGOCRYPT_EXPORT
bool
mongocrypt_kms_ctx_status (mongocrypt_kms_ctx_t *kms,
                           mongocrypt_status_t *status);


/**
 * Call when done handling all KMS contexts.
 *
 * @param[in] ctx The @ref mongocrypt_ctx_t object.
 *
 * @returns A boolean indicating success. If false, an error status is set.
 * Retrieve it with @ref mongocrypt_ctx_status
 */
MONGOCRYPT_EXPORT
bool
mongocrypt_ctx_kms_done (mongocrypt_ctx_t *ctx);


/**
 * Perform the final encryption or decryption.
 *
 * @param[in] ctx A @ref mongocrypt_ctx_t.
 * @param[out] out The final BSON to send to the server. The data viewed by @p
 * out is guaranteed to be valid until @p ctx is destroyed with @ref
 * mongocrypt_ctx_destroy.
 *
 * @returns a bool indicating success.
 */
MONGOCRYPT_EXPORT
bool
mongocrypt_ctx_finalize (mongocrypt_ctx_t *ctx, mongocrypt_binary_t *out);


/**
 * Destroy and free all memory associated with a @ref mongocrypt_ctx_t.
 *
 * @param[in] ctx A @ref mongocrypt_ctx_t.
 */
MONGOCRYPT_EXPORT
void
mongocrypt_ctx_destroy (mongocrypt_ctx_t *ctx);


#endif /* MONGOCRYPT_H */
