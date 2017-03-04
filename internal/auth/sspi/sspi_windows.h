#include <windows.h>

#include "sspi_call_windows.h"

SECURITY_STATUS SEC_ENTRY sspi_acquire_default_credentials_handle(CredHandle* cred_handle);

SECURITY_STATUS SEC_ENTRY sspi_acquire_credentials_handle(CredHandle* cred_handle, char* username, char* password);

SECURITY_STATUS SEC_ENTRY sspi_get_cred_name(CredHandle *cred_handle, char **out_name);

SECURITY_STATUS SEC_ENTRY sspi_free_credentials_handle(CredHandle* cred_handle);

SECURITY_STATUS SEC_ENTRY sspi_initialize_security_context(CredHandle* cred_handle, int has_context, CtxtHandle* context, PVOID buffer, ULONG buffer_length, PVOID* out_buffer, ULONG* out_buffer_length, char* target);

SECURITY_STATUS SEC_ENTRY sspi_delete_security_context(CtxtHandle* context);

SECURITY_STATUS SEC_ENTRY sspi_send_client_authz_id(CtxtHandle* context, PVOID* buffer, ULONG* buffer_length, char* user_plus_realm);