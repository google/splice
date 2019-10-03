# Splice

Splice is an infrastructure service which allows Windows instances to join a
Microsoft Active Directory domain without direct line of sight to a domain
controller. It also supports flexible user auth and complex join request
validation.

## Overview

In a traditional Windows domain, all clients must be "domain joined" during (or
after) imaging. The join establishes trust between the client and the domain
controllers, and can be used as the basis for long term remote management of the
device via a VPN, Microsoft DirectAccess, etc.

The join introduces a potential circular dependency for remote devices:

*   Trust must preceed remote access.
*   Access is required to establish trust.

The most basic solution is to always join clients on a network segment with
direct connectivity to the business domain, but this introduces limitations.
It's normally undesirable to expose domain controllers beyond the network
perimeter, and it may be logistically or functionally difficult to physically
connect every client to the domain network.

Splice addresses this dilemma by providing an intermediary broker for the domain
join operation. The Splice infrastructure spans the network perimeter, enabling
join requests to enter the network externally, and permits establishing domain
trust without ever requiring the client to directly contact a domain controller.
Once the join is complete, a management VPN can take over responsibility for the
device's lifecycle.

## Documentation

See the [Project Documentation](doc/index.md) for more information.

## Disclaimer

This is not an official Google product.
