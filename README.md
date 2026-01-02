# metaproxy

A flexible, protocol-agnostic proxy server that routes traffic based on connection metadata rather than deep packet inspection.

## What is metaproxy?

metaproxy is a lightweight TCP proxy system that makes routing decisions based on client-provided headers containing protocol, host, and port information. Think of it as nginx, but protocol-agnostic—it works equally well for SSH, game servers, custom protocols, or any TCP-based service.

### Key Features

- **Protocol-agnostic routing**: Route any TCP protocol based on metadata
- **Regex-based rules**: Flexible pattern matching for protocols, hosts, and ports
- **Cascading proxies**: Chain multiple metaproxy servers together with the `rec` keyword
- **Simple configuration**: Human-readable config files
- **Zero dependencies**: Single binary deployment

## Use Cases

- **Multi-tenant infrastructure**: Route different services to different backends based on subdomain
- **SSH without jump hosts**: Direct SSH connections through a single entry point without `-J` syntax
- **Game server routing**: Route Minecraft, game servers, or other UDP/TCP services by domain
- **Development environments**: Route traffic to different machines based on project or environment
- **Bypassing NAT/firewall limitations**: Expose services behind restrictive networks through a single public endpoint

## Installation

### Build from source

Clone the repository and run the build script:

```bash
git clone https://github.com/cactircool/metaproxy.git
cd metaproxy
chmod +x build.sh
./build.sh
```

The binaries will be placed in the `bin/` directory. You can then move them to your preferred location:

```bash
sudo mv bin/metaproxy /usr/local/bin/
sudo mv bin/mpc /usr/local/bin/  # Client wrapper (optional)
```

## Quick Start

### 1. Create a configuration file

Create a file called `proxy.cfg`:

```
25565 {
    # Route Minecraft traffic to local server
    [tcp;mc.example.com;25565] -> [192.168.1.10;25565]

    # Route SSH to different machines by subdomain
    [tcp;dev.example.com;22] -> [192.168.1.20;22]
    [tcp;prod.example.com;22] -> [192.168.1.30;22]

    # Cascade to another metaproxy server
    [tcp;internal.example.com;.*] -> rec [10.0.0.5;25565]

    # Explicitly reject certain patterns
    [tcp;blocked.example.com;.*] -> fail
}
```

### 2. Start the server

```bash
metaproxy server proxy.cfg
```

For verbose logging:

```bash
metaproxy server --verbose proxy.cfg
```

### 3. Connect from a client

Use the `metaproxy connect` command directly:

```bash
metaproxy connect tcp mc.example.com 25565
```

Or use the `mpc` wrapper for SSH (if you have the Python wrapper script):

```bash
mpc ssh user@dev.example.com
```

## Configuration Reference

### Basic Structure

```
PORT {
    [protocol;host;port] -> [forward_host;forward_port]
    [protocol;host;port] -> rec [forward_host;forward_port]
    [protocol;host;port] -> fail
}
```

### Configuration Fields

#### PORT (required)

The port on which this server instance listens. Must be between 0-65535.

You can specify multiple PORT blocks in a single config file to run multiple server instances:

```
8080 {
    [http;.*;.*] -> [localhost;80]
}

25565 {
    [tcp;.*;.*] -> [localhost;25565]
}
```

#### protocol

The protocol identifier to match. Can be:

- Empty string: matches any protocol (wildcard)
- Exact match: `tcp`, `udp`, `ssh`, etc.
- Regex pattern: `(ssh|sftp)`, `tcp.*`, etc.

#### host

The destination hostname or domain to match. Can be:

- Empty string: matches any host (wildcard)
- Exact match: `example.com`, `192.168.1.1`
- Regex pattern: `.*\.example\.com`, `dev-.*`

#### port

The destination port to match. Can be:

- Empty string: matches any port (wildcard)
- Exact match: `22`, `8080`
- Regex pattern: `(80|443)`, `80.*`

#### rec (optional)

Prefix keyword indicating the destination is also a metaproxy server. When specified, the original header is forwarded to the next server, allowing cascading proxies.

```
[tcp;internal.*;.*] -> rec [proxy2.example.com;25565]
```

#### forward_host (required)

The hostname or IP address to forward traffic to.

#### forward_port (required)

The port on the destination host to forward traffic to.

#### fail

Explicitly reject connections matching this pattern. The connection will be immediately closed.

```
[tcp;banned.example.com;.*] -> fail
```

### Rule Matching

- Rules are evaluated **top to bottom** within a PORT block
- The **first matching rule** is applied
- If no rules match, the connection is rejected (as if `fail` was used)
- Empty fields act as wildcards (match anything)
- All fields support regex patterns

### Comments

Comments start with `#` and continue to the end of the line:

```
25565 {
    # This is a comment
    [tcp;example.com;22] -> [localhost;22]  # Inline comment
}
```

**Note**: Comments are not allowed within `[]` brackets.

## Command Reference

### Server Mode

```bash
metaproxy server [flags] CONFIG_FILE [CONFIG_FILE...]
```

**Flags:**

- `-v, --verbose`: Enable verbose logging

**Examples:**

```bash
# Start with a single config
metaproxy server proxy.cfg

# Start with multiple configs
metaproxy server proxy1.cfg proxy2.cfg

# Verbose logging
metaproxy server --verbose proxy.cfg
```

### Client Mode

```bash
metaproxy connect [flags] PROTOCOL HOST PORT
```

**Flags:**

- `-p, --local-port PORT`: Specify local port (default: 0/wildcard)
- `-o, --output-port`: Output the client port as first 32 bits to stdout (mostly for wrapper's)

**Examples:**

```bash
# Basic connection
metaproxy connect tcp example.com 25565

# Specify local port
metaproxy connect tcp example.com 22 --local-port 5000

# With port output
metaproxy connect tcp example.com 8080 --output-port
```

## Architecture

### How It Works

1. **Client connects** to a metaproxy server
2. **Client sends header** containing protocol, host, and port information (JSON format)
3. **Server matches header** against configured rules (top to bottom)
4. **Server forwards traffic** to the matched destination
5. **Bidirectional proxying** begins between client and destination

### Header Format

The client sends a JSON header with the following structure:

```json
{
	"protocol": "tcp",
	"host": "example.com",
	"port": "22"
}
```

This header is automatically generated by the `metaproxy connect` command.

### Cascading Proxies

Using the `rec` keyword, you can chain multiple metaproxy servers:

```
Server A (public)          Server B (internal)        Final destination
     |                            |                          |
     |--[tcp;internal.*;.*]-->    |--[tcp;db.*;.*]-->   [database:5432]
                                  |--[tcp;web.*;.*]-->  [webserver:80]
```

Configuration for Server A:

```
8080 {
    [tcp;internal.*;.*] -> rec [serverB.local;8080]
}
```

Configuration for Server B:

```
8080 {
    [tcp;db.internal.com;.*] -> [10.0.0.10;5432]
    [tcp;web.internal.com;.*] -> [10.0.0.20;80]
}
```

## Advanced Examples

### Multi-environment SSH routing

Route SSH connections to different environments based on subdomain:

```
22 {
    [ssh;dev.*;22] -> [192.168.1.100;22]
    [ssh;staging.*;22] -> [192.168.1.101;22]
    [ssh;prod.*;22] -> [192.168.1.102;22]
    [ssh;.*;22] -> fail  # Reject unknown hosts
}
```

### Game server routing

```
25565 {
    # Route different Minecraft servers
    [tcp;survival.mc.example.com;.*] -> [10.0.1.10;25565]
    [tcp;creative.mc.example.com;.*] -> [10.0.1.11;25565]
    [tcp;minigames.mc.example.com;.*] -> [10.0.1.12;25565]

    # Default server
    [tcp;mc.example.com;.*] -> [10.0.1.10;25565]
}
```

### Protocol-based routing

```
8000 {
    # Route based on protocol identifier
    [ssh;.*;.*] -> [ssh-gateway;22]
    [http;.*;.*] -> [web-gateway;80]
    [postgres;.*;.*] -> [db-gateway;5432]
    [.*;.*;.*] -> fail  # Reject unknown protocols
}
```

## Limitations & Considerations

- **Client-side requirement**: Clients must use `metaproxy connect` or compatible wrappers
- **TCP only**: Currently only supports TCP connections
- **No encryption**: metaproxy itself doesn't provide encryption—use SSH tunnels or TLS for secure connections
- **Trust model**: The client specifies routing information, so only use in trusted environments or combine with authentication
- **HTTP/HTTPS**: Not ideal for browser traffic since browsers can't use the client wrapper

## Troubleshooting

### Port already in use

```
failed to start server on port 8080: cannot bind to port 8080
```

**Solution**: Another service is using that port. Choose a different port or stop the conflicting service.

### Connection refused

**Solution**: Check that:

1. The metaproxy server is running
2. Firewall rules allow the connection
3. The destination service is running and accessible from the proxy server

### No matching rule

```
unmapped header detected, forcing sudoku
```

**Solution**: The connection didn't match any configured rules. Add a rule for this connection or add a wildcard catch-all rule.

### Enable verbose logging

```bash
metaproxy server --verbose proxy.cfg
```

This will show connection attempts and routing decisions.

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests on GitHub.

## License

See the [LICENSE](LICENSE) file for details.

## Acknowledgments

Built with:

- [Cobra](https://github.com/spf13/cobra) for CLI parsing
- Go standard library for networking

---

**Author**: cactircool  
**Repository**: https://github.com/cactircool/metaproxy
