# Splice App

Splice App runs in your Google Cloud Platform (GCP) project. It serves as the
intermediary for the transport of Domain Join metadata between the CLI and your
SpliceD server inside your perimeter.

## Project Selection

Splice can run in your existing GCP organization, either in its own project, or
within a project you already run.

1.  Go to https://console.cloud.google.com
1.  Select your project, or create a new one if necessary.

### Account Setup

1.  Go to `IAM & Admin`
1.  In the IAM section
    *   set your project owner
        *   Project > Owner
1.  Go to Service Accounts
    *   Create a service account to be used by SpliceD.
        *   Assign the role Datastore > Cloud Datastore Owner

#### Account Credentials

Cloud console will provide a JSON encoded credential file for service accounts,
which can be used to authenticate an application from Windows.

WARNING: The credential file is sensitive, and allows anyone to impersonate the
role account. Never store the credential file anywhere it can be accessed by
non-administrators.

1.  From the IAM & Admin panel, go to Service Accounts.
1.  Select the more options menu for the SpliceD service account and select
    Create Key. Leave type as JSON.

Service account keys can be deleted by clicking the small trash can icon next to
the Key ID string. This will invalidate previously issued keys, requiring new
keys to be distributed.

### Pub/Sub Setup

Pub/Sub notifies SpliceD of waiting join requests. In pull mode, this does not
require AppEngine to have any inbound access to the network perimeter.

1.  Go to Pub/Sub
1.  Click Create Topic
    *   Name: `requests`
1.  Select the topic and create a New Subscription
    *   Name: `spliced`
    *   Type: `pull`
1.  Select the subscription and open Permissions
    *   Add the role user created during *Account Setup* to:
        *   Pub/Sub Viewer
        *   Pub/Sub Subscriber

### Datastore Setup

The datastore maintains all state for active requests.

1.  Go to Datastore
1.  Create Entity
1.  Create an Entity
    *   Kind: "`RequestList`"
    *   Key identifier: `Custom name`: "`default`"

### Deployment

Splice App is written in Go. See "[Deploying a Go
App](https://cloud.google.com/appengine/docs/standard/go/tools/uploadinganapp)"
for information on how to deploy splice to App Engine in your project.

### Project Allowlist

When used with the -gce flag, the Splice CLI will submit GCE [identity metadata](https://cloud.google.com/compute/docs/instances/verifying-instance-identity) with its request to Splice App for validation. This allows splice to restrict incoming requests to a verifiable list of authorized App Engine projects.

The project allowlist is contained in app.yaml.


