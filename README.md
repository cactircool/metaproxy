# metaproxy

## ramble

Ever been locked into AT&T as your internet provider, while simultaneously being a poor college student just trying to get cheap publicly routable computers without slow tunneling?

You might be saying at this point - "Just use IPv6 prefix delegation and delegate a prefix to use on your devices and BAM, now they're publicly routable." And to that I say, AT&T doesn't support it. Worse yet, the diagnostic tools give mixed results (for me at least) of exactly ONE result that indicates prefix delegation is enabled, giving me false hope. Then I'm thrust into a day long rabbit hole trying my hardest to exploit that one ray of hope to find out its impossible.

Anyway, the main usecase I wanted was to map a bunch of subdomains of a domain I had bought to specific small laptops and raspberry pis I had lying around, effectively allowing me to use them as web servers AND ssh into them. With tools like nginx, the web server one is actually pretty easy with repeated forwarding of traffic from one entry point server that does have the public ipv6 address.

However, what about ssh? I have to use the ugly ssh jump syntax and I would rather die. So instead, what if ssh sent a header over tls with the source information the client entered into the command? In that case, you could forward traffic based on domain and protocol. Basically, its like nginx but much more versatile, supporting many protocols and switching over the method by which the client connected.

Only syntactic caveat, the client has to wrap their command with `mpc` like so:

```
mpc ssh -p MP_SERVER_PORT user@address
```

But I'd much rather use this over that dumbass -J jump syntax where I have to specify multiple addresses AND REMEMBER THEM EACH TIME.

Other caveats include, the client must use the mpc program to proxy the traffic, which means general purpose traffic like with HTTP/HTTPS where I can't enforce that the browser wraps its traffic doesn't get the benefits of this program.

Best for protocols where me or people I trust are the clients and I set up the server.

## build from source

There is conveniently a script called `mp-setup.py` in the root directory of this project. Simply run this script with sudo:
`sudo python3 mp-setup.py` and the client and server scripts will be available to you and added to `/usr/local/bin`. There are no dependencies other than that, so removing the binaries is enough to purge the program from your system.

By the way, this README.md assumes the programs are called `mp` for the raw proxy/server and `mpc` for the proxy wrapper (client use). However, the mp-setup.py lets you name them whatever you want so go nuts.

`mp-setup.py` will end up being platform independent eventually, but for now works on linux as far as I know. Eventually it will also automatically set up services like through Ubuntu's systemctl so starting persistent servers is easy.

## set up a server

You may be asking - how do you set up a server?

Great question, basically via VERY simple configuration file(s).

1. Create a configuration file
2. Populate it in the following format:

```
PORT { # Comment
	[protocol;host;port] -> rec [forward_host;forward_port] # Comment
	[protocol2;host2;port2] -> [forward_host2;forward_port2]
	[a;b;c] -> fail
	...
}

...
```

3. Run `mp server [CONFIG_FILES...]` to get a server instance running

## explaining config files

### PORT

- REQUIRED: A positive integer [0, 65335] The port that the server instance will be running on. Specifying multiple PORT blocks simply causes multiple server instances to start up on all ports specified using the specific config for that port.

### protocol

- The protocol that metaproxy will look for when matching with this rule. Leaving it unspecified acts as a wild card. Regex syntax is allowed.

### host

- The host that metaproxy will look for when matching with this rule. Leaving it unspecified acts as a wild card. Regex syntax is allowed.

## port

- If you also want to select based on what port from the machine that connected used to connect, this also lets you do that. This is also regex, so regex rules apply. Empty will be treated as an automatic match.

### rec

- Specifies if the server being forwarded to is also a metaproxy server. If so, the header that this server recieved will be forwarded to the forward server before the protocol data is routed. If unspecified, this server will simply forward all data blindly into the forward address.

### forward_host

- REQUIRED: The host to forward traffic to.

### forward_port

- REQUIRED: The port on the host to forward traffic to.

### fail

- Used to explicitly declare a pattern as immediate failure and closing of the connection with the client.

### notes

- Rules are applied top down within a block until one matches the connection header. A header that doesn't match any rule will fail as if the fail keyword was used.

- Comments are not allowed within `[]`. But anywhere else is fine since they are treated as basic whitespace.

## connecting to a server from a client

Imagine a protocol like ssh or telnet (atm only ssh is supported since I don't really use other protocols that often, but I'll add them later. Shouldn't be too hard idt). Now write the command for that protocol but just add `mpc` before the command.

```
mpc ssh -p MP_SERVER_PORT user@address
```

If you set up the server on port 22:

```
mpc ssh user@address
```

## help

Use the any of `--help`, `-help`, `-h`, or `--h` flags to see a usage help screen.

## closing

That's about everything!

This lets me better connect my devices with just one point of entry. I really hope this isn't already an existing tool, cuz I will be sad then. :]

I'm actually very proud of this small piece of software, its kinda nice

## TODO

- Factor out mpc into another repo and use go again.
- Use cobra for the cli parsing instead of the built in flag library so nested commands are nicer.
- Remove mp-setup.py I guess.
- Work on getting package manager support for both packages, or create a seperate init script for each one.
