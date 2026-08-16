# Socket

**Socket** is an open-source encrypted bind shell that uses mutual TLS (mTLS) for secure and authenticated client-server communication.

The Socket server is designed to run on Linux systems. Clients can connect to it using standard tools such as OpenSSL.

# Features

- Secure and encrypted communication over mTLS
- Mutual authentication between client and server
- Support for multiple simultaneous clients
- Server-side interactive PTY shell
- Configurable listening address and port

# Installation

1. Clone the repository:

```bash
git clone https://github.com/Yakziee/Socket.git
```

2. Enter the project directory:

```bash
cd Socket
```

3. Build the program:

```bash
go build -o Socket
```

4. Run the server:

```bash
./Socket <port>
```

# Usage

When the server is started for the first time, the files `ca.crt`, `client.crt`, and `client.key` are generated in the current working directory.

These files are required when connecting to the server. For example, using OpenSSL:

```bash
openssl s_client -connect <serverIP>:<port> -cert client.crt -key client.key -CAfile ca.crt
```
