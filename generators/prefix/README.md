# Splice Prefix Name Generator

The Prefix name generator (generator ID: "prefix") generates hostnames using a
fixed prefix string plus one or more randomly generated characters.

Prefix takes two inputs, the prefix string and a total length. For a prefix of
"EXAMPLE-" and a length of 12, each name will be akin to "EXAMPLE-A1B2" or
"EXAMPLE-B8CX".

The charset used for the random string can be modified by changing the `charset`
const inside the package. By default it will use A-Z0-9.

For a Prefix of length N, Length must be at minimum N+1. Note that because the
string generated is purely random, collisions are possible and it may
occasionally be necessary to regenerate a name to avoid a collision. Increasing
the size of the random segment should minimize the occurrence of collisions, but
will never completely eliminate them.

## Configuration

The Prefix generator is configured via the registry.

*   `SOFTWARE\Splice\Generators\prefix\Prefix`
    *   Type: string
    *   Value: The static prefix string to use with all names
*   `SOFTWARE\Splice\Generators\prefix\Length`
    *   Type: DWORD
    *   Value: The desired name length, including prefix
