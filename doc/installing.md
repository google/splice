# Installing Splice

## Requirements

*   A Go compiler to build the executables
*   A [Google Cloud](https://cloud.google.com) project
*   At least one domain joined Windows host inside the domain perimeter

## Infrastructure Overview

The Splice design consists of three components, built on top of Google Cloud
infrastructure services: Splice CLI, Splice App, and SpliceD.

The **[SpliceD](../spliced/README.md)** native Windows service is installed
inside the domain perimeter on one or more Windows servers. SpliceD is
responsible for making the necessary API calls to handle the on-network portion
of the join.

The **[Splice CLI](../cli/README.md)** is a standalone command line application
that will run within the unjoined Windows client. It gathers the necessary
information to initiate the domain join and posts a request to the Splice App.

The **[Splice App](../appengine/README.md)** is a Google AppEngine application
which runs on Google Cloud. The App receives join requests, performs internal
request validation, and places the request in a datastore for processing.
SpliceD is notified of a pending request, retrieves it from cloud storage, and
performs the join. The join result propagates back out via the App to the CLI,
where a final API call installs the join metadata. Following a reboot, the host
is now joined to the domain.

## Cloud Setup

1.  Follow the steps in the [Splice App README](../appengine/README.md) to
    configure your Google Cloud project.
1.  Deploy the Splice App binary to AppEngine in your project.

## SpliceD Setup

Review the [SpliceD README](../appengine/README.md) for detailed information.

1.  Build and configure a dedicated Windows host to run SpliceD. This instance
    must already be domain joined.
1.  Copy the designated role account credentials from the Cloud Project to the
    server.
1.  Build the spliced.exe executable and install on the server.
1.  On the server, use the `spliced.exe configure`
    [command line](../spliced/README.md#config) to add the daemon settings to
    the registry.
1.  Register `spliced.exe` as a system service:

    ```
    New-Service -Name SpliceD -BinaryPathName 'C:\Program Files\SpliceD\spliced_svc.exe' -Description 'The SpliceD domain joiner.' -StartupType Automatic
    ```

1.  Start the service, and confirm in the host Event Log that it is waiting for
    requests.

## Client Setup

1.  Build the cli Go binary and copy to the Windows host, or bake into the
    installer.
1.  Run with the [appropriate flags](../cli/README.md#usage) when ready to join.
