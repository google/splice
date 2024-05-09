# Splice CLI

The Splice CLI runs on the machine to be joined, initiating the join request and
installing the resulting metadata.

## Usage {#usage}

cli.exe is configured via runtime flags.

`cli -name myname -server https://splice.example.com`

*   **-name**: (required) The host name to be requested for the join.
*   **-server**: (required) The appengine server url hosting the Splice app.
*   **-encrypt**: (optional) Encrypt metadata in transit. See
    [encryption](#encryption).
*   **-cert_issuer**: (optional) The certificate issuer to look for when
    locating the host certificate to use for encryption. Requires '-encrypt'.
*   **-cert_container**: (optional) The container name of the private key that
    is associated with the host certificate. Requires '-encrypt'.
*   **-generate_cert**: (optional) Generates a temporary self-signed certificate
    to use for encryption, in lieu of a host certificate. Requires '-encrypt'.
*   **-really_join**: (optional) Specifies of the domain join operation should
    be finalized, defaults to false.
*   **-unattended**: (optional) Makes requests using unattended mode. Requires
    the gce flag.
*   **-gce**: (optional) Includes GCE metadata with the request. Only used by
    the unattended flag.
*   **-verbose**: (optional) Include verbose output during the offline domain
    join.

## Feature Detail

### encryption {#encryption}

Join metadata is considered sensitive material, and should be kept well secured.
By default, the metadata is encrypted in transit between the SpliceD backend and
the App Datastore, on disk while resident in the Datastore, and in transit
between the App and CLI (assuming https).

In the above scenario, the metadata would be visible to users and accounts with
read access to the App Datastore.

For an additional layer of security the `-encrypt` flag will instruct Splice CLI
to provide a public key, which the SpliceD backend can use to encrypt the
metadata while in transit.

1.  Splice CLI locates (or generates) a certificate.
1.  The certificate is included in the join request to the App server.
1.  SpliceD retrieves the certificate when the request is accepted.
1.  If the join succeeds, the metadata is encrypted:
    1.  The metadata blob is encrypted using an temporary AES key.
    1.  The AES key is encrypted using the public key from the CLI.
1.  Both the key and the metadata are returned to the datastore.
1.  The CLI decrypts the AES key using the local certificate, and decrypts the
    metadata using the resulting AES key.

Note: If the `encrypt_blob` setting is configured in SpliceD, encryption via the
`-encrypt` flag is required for the request to complete successfully.

#### Host Certificates

Metadata encryption with host certificates is supported by using the
`-cert_issuer` and `-cert_container` flags together with the `-encrypt` flag.
When used, the CLI will search for a certificate from the specified issuer and a
private key in a container of the specified name. This certificate is provided
to the App, which passes it down to SpliceD for metadata encryption. When the
encrypted metadata is returned, the CLI decrypts the metadata using the private
key of the host certificate.

THe use of hardware (TPM) backed certificates for metadata encryption are
natively supported through the use of
[certtostore](https://github.com/google/certtostore) and the
[Microsoft Crypto Next Generation API](https://msdn.microsoft.com/en-us/library/windows/desktop/aa376210\(v=vs.85\).aspx).

#### Temporary Certificates

When used with both the `-generate_cert` flag, Splice CLI will generate a
temporary self-signed certificate in memory specifically for the purpose of
metadata encryption.
