// Code adapted from the NodeJS kerberos library:
// 
//   https://github.com/christkv/kerberos/tree/master/lib/win32/kerberos_sspi.h
//
// Under the terms of the Apache License, Version 2.0:
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
#ifndef SSPI_CALL_WINDOWS_H
#define SSPI_CALL_WINDOWS_H

#define SECURITY_WIN32 1  /* Required for SSPI */

#include <windows.h>
#include <sspi.h>

int load_secur32_dll();

SECURITY_STATUS SEC_ENTRY call_sspi_encrypt_message(
    PCtxtHandle phContext, 
    unsigned long fQOP, 
    PSecBufferDesc pMessage, 
    unsigned long MessageSeqNo
);

typedef DWORD (WINAPI *encryptMessage_fn)(PCtxtHandle phContext, ULONG fQOP, PSecBufferDesc pMessage, ULONG MessageSeqNo);

SECURITY_STATUS SEC_ENTRY call_sspi_acquire_credentials_handle(
    LPSTR pszPrincipal,             // Name of principal
    LPSTR pszPackage,               // Name of package
    unsigned long fCredentialUse,   // Flags indicating use
    void *pvLogonId,                // Pointer to logon ID
    void *pAuthData,                // Package specific data
    SEC_GET_KEY_FN pGetKeyFn,       // Pointer to GetKey() func
    void *pvGetKeyArgument,         // Value to pass to GetKey()
    PCredHandle phCredential,       // (out) Cred Handle
    PTimeStamp ptsExpiry            // (out) Lifetime (optional)
);

typedef DWORD (WINAPI *acquireCredentialsHandle_fn)(
    LPSTR pszPrincipal, LPSTR pszPackage, unsigned long fCredentialUse,
    void *pvLogonId, void *pAuthData, SEC_GET_KEY_FN pGetKeyFn, void *pvGetKeyArgument,
    PCredHandle phCredential, PTimeStamp ptsExpiry
);

SECURITY_STATUS SEC_ENTRY call_sspi_query_credentials_attributes(
    PCredHandle phCredential,       // Handle to the credentials to be queried
    unsigned long ulAttribute,      // Specifies the attribute to be queried
    void *pBuffer                   // Buffer for attributes
);

typedef DWORD (WINAPI *queryCredentialsAttributes_fn)(
    PCredHandle phCredential, unsigned long ulAttribute, void *pBuffer);

SECURITY_STATUS SEC_ENTRY call_sspi_free_credentials_handle(
  PCredHandle phCredential
);

typedef DWORD (WINAPI *freeCredentialsHandle_fn)(
    PCredHandle phCredential);

SECURITY_STATUS SEC_ENTRY call_sspi_initialize_security_context(
    PCredHandle phCredential,       // Cred to base context
    PCtxtHandle phContext,          // Existing context (OPT)
    LPSTR pszTargetName,            // Name of target
    unsigned long fContextReq,      // Context Requirements
    unsigned long Reserved1,        // Reserved, MBZ
    unsigned long TargetDataRep,    // Data rep of target
    PSecBufferDesc pInput,          // Input Buffers
    unsigned long Reserved2,        // Reserved, MBZ
    PCtxtHandle phNewContext,       // (out) New Context handle
    PSecBufferDesc pOutput,         // (inout) Output Buffers
    unsigned long *pfContextAttr,   // (out) Context attrs
    PTimeStamp ptsExpiry            // (out) Life span (OPT)
);

typedef DWORD (WINAPI *initializeSecurityContext_fn)(
    PCredHandle phCredential, PCtxtHandle phContext, LPSTR pszTargetName, unsigned long fContextReq,
    unsigned long Reserved1, unsigned long TargetDataRep, PSecBufferDesc pInput, unsigned long Reserved2,
    PCtxtHandle phNewContext, PSecBufferDesc pOutput, unsigned long *pfContextAttr, PTimeStamp ptsExpiry);

SECURITY_STATUS SEC_ENTRY call_sspi_query_context_attributes(
    PCtxtHandle phContext,          // Context to query
    unsigned long ulAttribute,      // Attribute to query
    void *pBuffer                   // Buffer for attributes
);

typedef DWORD (WINAPI *queryContextAttributes_fn)(
    PCtxtHandle phContext, unsigned long ulAttribute, void *pBuffer);

SECURITY_STATUS SEC_ENTRY call_sspi_delete_security_context(
  PCtxtHandle phContext
);

typedef DWORD (WINAPI *deleteSecurityContext_fn)(
    PCtxtHandle phContext);

SECURITY_STATUS SEC_ENTRY call_sspi_free_context_buffer(
  void* pBuffer
);

typedef DWORD (WINAPI *freeContextBuffer_fn)(
    void *pBuffer);

#endif // SSPI_CALL_WINDOWS_H