# Splice Name Generators

Each domain join requires the assignment of a host name. By default, Splice
allows clients to request their desired hostname via the CLI `-name` flag. In
some scenarios it may be preferable to have the infrastructure generate a name
for the join process instead, for example in the case of headless installations,
or where a specific naming scheme is desired. Splice Name Generators provide a
mechanism for automatic generation of host names during the join process.

Generators are a feature of SpliceD, the backend domain join daemon. SpliceD can
support multiple Generators at once, with each identified by a unique name.
Rather than requesting a hostname, the client request includes the name of the
desired generator. When the request is serviced, the generator uses its internal
logic to produce a candidate name, after which the join proceeds.

## Available Generators

### Prefix

The Prefix generator produces a name with a fixed prefix string followed by
random characters. For example, if this generator is configured with the prefix
"EXAMPLE-" and the length 12, it will generate names such as "EXAMPLE-6V9A",
with a 4 character random suffix.

[Documentation](../generators/prefix/README.md)

## Building Generators

Generators are pluggable Go packages that extend SpliceD. Building a new
generator requires creating the generator package, linking it into the main
SpliceD binary, and configuring the SpliceD service to enable it.

Generators can have virtually any logic they like in them for the purposes of
producing names. However they also need to include a few core behaviors that
make them interoperate correctly with the rest of SpliceD. Let's look at an
extremely simplistic example generator that generates names based on a (very
impractical) internal counter.

`splice/generators/example/example.go`:

```
package example

import (
    "github.com/google/splice/generators"
  )

type gen struct {
  configured bool
  counter    int
}

func init() {
    generators.Register("example", &gen{})
}
```

This file creates a Go package named "example" and registers it with SpliceD
using the same name. The init() function is called while Go is first loading the
package. The use of `generators.Register` registers the name "example", and
associates it with an instance of the `gen` struct.

```
func (p *gen) Configure() error {
  p.counter = 0
    p.configured = true
    return nil
}
```

Each generator should have a `func Configure() error` member. This function
should perform any steps necessary to configure the generator for use. This
might include reading settings from the registry or from a file on disk. This
will be called at least once on startup, but may be called multiple times as a
way of refreshing settings later.

```
func (p *gen) Generate(input []byte) (string, error) {
    if !p.configured {
        return "", generators.ErrNotConfigured
    }
  p.counter+=1
  return fmt.Sprintf("EXAMPLE-%d", p.counter), nil
}
```

The `func Generate([]byte) (string, error)` member is responsible for performing
the name generation action, and is invoked every time a join request is received
for this generator.

The `input` []byte slice is optional. It allows the client to pass arbitrary
data to the generator which may be useful in certain naming schemes. Note that
generator authors should be extremely careful with this input, as it constitutes
a potentially unsanitized user input. It may be necessary to extend either the
CLI and/or the App to better support the use of the input fields.

If a string is returned with no error, SpliceD will attempt to perform a join
using the string. If an error is returned, the join will fail and the client
will receive the failure in return.

Finally, the generator package must be linked to SpliceD by importing it into
the SpliceD codebase.

`github.com/google/splice/spliced/service_windows.go:`

```
import(
  _ "github.com/google/splice/generators/example"
)
```

Once SpliceD is rebuilt and configured, clients should be able to request names
from this generator by passing `-generator_id=example` in place of `-name`.
