#include <string.h>
#include <stdio.h>
#include "gssapi_wrapper.h"

void gssapi_print_error(
    OM_uint32 status_code
)
{
    OM_uint32 message_context;
    OM_uint32 maj_status;
    OM_uint32 min_status;
    gss_buffer_desc status_string;

    message_context = 0;

    do {
        maj_status = gss_display_status(&min_status, status_code, GSS_C_GSS_CODE, GSS_C_NO_OID, &message_context, &status_string);
        printf("%.*s\n", (int)status_string.length, (char *)status_string.value);
        gss_release_buffer(&min_status, &status_string);
    } while (message_context != 0);
}

int gssapi_client_init(
    gssapi_client_state *client,
    char* spn,
    char* username
)
{
    client->cred = GSS_C_NO_CREDENTIAL;
    client->ctx = GSS_C_NO_CONTEXT;

    gss_buffer_desc spn_buffer = GSS_C_EMPTY_BUFFER;

    spn_buffer.value = spn;
    spn_buffer.length = strlen(spn);
    client->maj_stat = gss_import_name(&client->min_stat, &spn_buffer, GSS_C_NT_HOSTBASED_SERVICE, &client->spn);

    if (GSS_ERROR(client->maj_stat)) {
        return GSSAPI_ERROR;
    }

    if (username) {
        gss_buffer_desc name_buffer = GSS_C_EMPTY_BUFFER;
        name_buffer.value = username;
        name_buffer.length = strlen(username);
        gss_name_t name;

        client->maj_stat = gss_import_name(&client->min_stat, &name_buffer, GSS_C_NT_USER_NAME, &name);
        if (GSS_ERROR(client->maj_stat)) {
            return GSSAPI_ERROR;
        }

        client->maj_stat = gss_acquire_cred(&client->min_stat, name, GSS_C_INDEFINITE, GSS_C_NO_OID_SET, GSS_C_INITIATE, &client->cred, NULL, NULL);
        if (GSS_ERROR(client->maj_stat)) {
            return GSSAPI_ERROR;
        }

        client->maj_stat = gss_release_name(&client->min_stat, &name);
        if (GSS_ERROR(client->maj_stat)) {
            return GSSAPI_ERROR;
        }
    }

    return GSSAPI_OK;
}

int gssapi_client_destroy(
    gssapi_client_state *client
)
{
    OM_uint32 maj_stat, min_stat;
    if (client->ctx != GSS_C_NO_CONTEXT) {
        maj_stat = gss_delete_sec_context(&min_stat, &client->ctx, GSS_C_NO_BUFFER);
    }

    if (client->spn != GSS_C_NO_NAME) {
        maj_stat = gss_release_name(&min_stat, &client->spn);
    }

    if (client->cred != GSS_C_NO_CREDENTIAL) {
        maj_stat = gss_release_cred(&min_stat, &client->cred);
    }
}

int gssapi_client_get_username(
    gssapi_client_state *client,
    char** username
)
{
    gss_name_t name = GSS_C_NO_NAME;

    client->maj_stat = gss_inquire_context(&client->min_stat, client->ctx, &name, NULL, NULL, NULL, NULL, NULL, NULL);
    if (GSS_ERROR(client->maj_stat)) {
        return GSSAPI_ERROR;
    }

    gss_buffer_desc name_buffer;
    client->maj_stat = gss_display_name(&client->min_stat, name, &name_buffer, NULL);
    if (GSS_ERROR(client->maj_stat)) {
        if (name_buffer.value) {
            gss_release_buffer(&client->min_stat, &name_buffer);
        }
        gss_release_name(&client->min_stat, &name);
        return GSSAPI_ERROR;
    }

	*username = malloc(name_buffer.length+1);
	memcpy(*username, name_buffer.value, name_buffer.length+1);

    gss_release_buffer(&client->min_stat, &name_buffer);
    gss_release_name(&client->min_stat, &name);
    return GSSAPI_OK;
}

int gssapi_client_wrap_msg(
    gssapi_client_state *client,
    void* input,
    size_t input_length,
    void** output,
    size_t* output_length 
)
{
    gss_buffer_desc input_token = GSS_C_EMPTY_BUFFER;
    gss_buffer_desc output_token = GSS_C_EMPTY_BUFFER;

    input_token.value = input;
    input_token.length = input_length;

    client->maj_stat = gss_wrap(&client->min_stat, client->ctx, 0, GSS_C_QOP_DEFAULT, &input_token, NULL, &output_token);

    if (output_token.length) {
        *output = malloc(output_token.length);
        *output_length = output_token.length;
        memcpy(*output, output_token.value, output_token.length);

        client->maj_stat = gss_release_buffer(&client->min_stat, &output_token);
    }

    if (GSS_ERROR(client->maj_stat)) {
        gssapi_print_error(client->maj_stat);
        return GSSAPI_ERROR;
    }

    return GSSAPI_OK;
}

int gssapi_init_sec_context(
    gssapi_client_state *client,
    void* input,
    size_t input_length,
    void** output,
    size_t* output_length
)
{
    gss_buffer_desc input_token = GSS_C_EMPTY_BUFFER;
    gss_buffer_desc output_token = GSS_C_EMPTY_BUFFER;

    if (input) {
        input_token.value = input;
        input_token.length = input_length;
    }

    client->maj_stat = gss_init_sec_context(
        &client->min_stat,
        client->cred,
        &client->ctx,
        client->spn,
        GSS_C_NO_OID,
        GSS_C_MUTUAL_FLAG | GSS_C_SEQUENCE_FLAG,
        0,
        GSS_C_NO_CHANNEL_BINDINGS,
        &input_token,
        NULL,
        &output_token,
        NULL,
        NULL
    );

    if (output_token.length) {
        *output = malloc(output_token.length);
        *output_length = output_token.length;
        memcpy(*output, output_token.value, output_token.length);

        OM_uint32 maj_stat, min_stat;
        maj_stat = gss_release_buffer(&min_stat, &output_token);
        if (GSS_ERROR(maj_stat)) {
            client->maj_stat = maj_stat;
            client->min_stat = min_stat;
            return GSSAPI_ERROR;
        }
    }

    if (GSS_ERROR(client->maj_stat)) {
        return GSSAPI_ERROR;
    } else if (client->maj_stat == GSS_S_CONTINUE_NEEDED) {
        return GSSAPI_CONTINUE;
    }

    return GSSAPI_OK;
}

