# SpliceD

SpliceD (Splice Daemon) lives inside the perimeter of the Windows domain and
handles new host join requests.

## Authentication

SpliceD uses [Google Application Default
Credentials](https://developers.google.com/identity/protocols/application-default-credentials)
to interact with the Cloud Datastore. The daemon requires a provisioned service
account with datastore privileges.

### Account Setup

1.  In the Cloud Project, go to IAM & Admin
1.  Go to Service Accounts
1.  Create a new service account
    *   Select the option to *Furnish a new private key*
1.  Assign the role Datastore > Cloud Datastore Owner to the new account.

#### Account Credentials

The Cloud control panel will provide a JSON encoded credential file for the new
account, which can be used to authenticate an application from Windows.

WARNING: The credential file is sensitive, and allows anyone to impersonate the
role account. Never store it where it can be exposed to non-administrators, and
scrub it from any intermediary storage. When in doubt, you can delete the
existing key and issue a new one in the service account UI.

To install the credentials on the SpliceD server:

1.  Copy the service account .json file to the server filesystem. (Ex:
    C:\ProgramData\SpliceD)
1.  Configure a new system environment variable with the path to the file.
    *   Name: `GOOGLE_APPLICATION_CREDENTIALS`
    *   Value: \[Path to .json\]
1.  Consider restricting the filesystem ACLs on the credential file to limit
    access to the service account.

## Configuration {#config}

Configuration is handled via the registry. Before enabling the spliced service,
run spliced.exe from the command line to configure the application.

    spliced configure -domain "domain.example.com" -instance "spliced123" -project "example-cloud-project" -topic "subscription1"

You can modify settings by re-running the configure command with one or more
parameters and restarting the service.

### Detail

*   HKLM\SOFTWARE\Splice\spliced
    *   Name: domain
        *   Type: REG_SZ
        *   Data: The name of the domain to be joined
            *   Example: 'domain.example.com'
    *   Name: instance
        *   Type: REG_SZ
        *   Data: A unique name for this backend.
            *   Example: 'spliced123'
    *   Name: project
        *   Type: REG_SZ
        *   Data: The name of the Google Cloud project.
            *   Example: 'example-cloud-project'
    *   Name: topic
        *   Type: REG_SZ
        *   Data: The name of the PubSub subscription topic.
            *   Example: 'subscription1'
    *   Name: encrypt_blob
        *   Type: REG_DWORD
        *   Data: 1 to enable; 0 to disable
        *   Default: enabled
    *   Name: verify_certs
        *   Type: REG_DWORD
        *   Data: 1 to enable; 0 to disable
        *   Default: enabled
    *   Name: ca_root_url
        *   Type: REG_SZ
        *   Data: The root URL of the certificate authority to use for
            certificate verification. Required when `verify_certs = 1`. See also
            ca_cert_path.
            *   Example: 'https://my.rootca.com/'
    *   Name: ca_cert_path
        *   Type: REG_SZ
        *   Data: The path under ca_root_url where the public certificates to be
            used during verification are stored. This path is appended to
            ca_root_url and used to locate the issuing certificate for
            verification. Required when `verify_certs = 1`.
            *   Example: 'some/ca/path'
    *   Name: permit_reuse
        *   Type: REG_DWORD
        *   Data: 1 to enable; 0 to disable

## Feature Detail

### encrypt_blob

If set, SpliceD will attempt to RSA-encrypt the metadata blob before uploading
it. This requires the request to include a certificate from the Splice client.

If no certificate is provided by the client, the join will fail.

### permit_reuse

The NetProvisionComputerAccount function supports the
`NETSETUP_PROVISION_REUSE_ACCOUNT` option. If enabled, the join will attempt to
reuse an existing account of the same name, should one exist.

Reference:
https://msdn.microsoft.com/en-us/library/windows/desktop/dd815228(v=vs.85).aspx

## Logging

SpliceD will log to the Application Event Log under the source name `SpliceD`.

## Debugging

Check the Event Log for any errors caught by the application.

The API calls used to perform the domain join will cause Windows to log to the
NetSetup.log file located at `C:\Windows\debug\NetSetup.LOG`.

### netProvisionComputerAccount failed to return successfully (XXXX,The operation completed successfully.)

The netProvisionComputerAccount api call will return a System Error Code if the
join fails, which will be logged to the Event Log. The error code (in hex, XXXX
above) can be translated to a specific condition by reviewing Microsoft's error
handling reference.

*   08b0: Account already exists. See `permit_reuse`.
*   216D: Joiner's machine account quota has been exceeded.

https://msdn.microsoft.com/en-us/library/windows/desktop/ms681381(v=vs.85).aspx

## Machine Account Quota

Active Directory gives each existing account a fixed quota for the number of new
machine accounts that can be joined using the existing account's credentials.
(If SpliceD is running as the local system account, this quota applies to the
host's machine account.) The quota defaults to 10, and is configured by the
`MS-DS-Machine-Account-Quota` attribute on the domain.

If the quota is unchanged, SpliceD will begin returning an error after its host
account reaches the join quota.

There are multiple ways to adjust the domain quota. One of the most practical is
to apply a custom permission set on the OU where SpliceD will create the machine
accounts.

1.  Open Active Directory Users and Computers
1.  On the OU receiving new computer accounts, go to Properties
1.  Go to Security tab
1.  Add either:
    *   The domain user the SpliceD service will run as or
    *   The machine account(s) where SpliceD is installed or
    *   A security group containing the domain users or machine accounts
1.  Using the Advanced option, grant the accounts the Create Computer Object
    privilege.
    *   You may revoke any additional privileges.

https://msdn.microsoft.com/en-us/library/ms678639(v=vs.85).aspx
