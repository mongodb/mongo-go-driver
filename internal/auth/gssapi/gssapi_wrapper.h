#include <stdlib.h>
#include <gssapi/gssapi.h>
#include <gssapi/gssapi_krb5.h>

#define GSSAPI_OK 0
#define GSSAPI_CONTINUE 1
#define GSSAPI_ERROR 2

typedef struct {
    gss_name_t spn;
    gss_cred_id_t cred;
    gss_ctx_id_t ctx;

    OM_uint32 maj_stat;
    OM_uint32 min_stat;
} gssapi_client_state;

int gssapi_client_init(
    gssapi_client_state *client,
    char* spn,
    char* username
);

int gssapi_client_destroy(
    gssapi_client_state *client
);

int gssapi_client_get_username(
    gssapi_client_state *client,
    char** username
);

int gssapi_client_wrap_msg(
    gssapi_client_state *client,
    void* input,
    size_t input_length,
    void** output,
    size_t* output_length 
);

int gssapi_init_sec_context(
    gssapi_client_state *client,
    void* input,
    size_t input_length,
    void** output,
    size_t* output_length
);
