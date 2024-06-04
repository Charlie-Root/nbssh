# nbssh

`nbssh` is a command-line tool written in Go that looks up the primary IP address of a device or virtual machine in Netbox via its API and initiates an SSH connection to the host.

## Features

- Look up a primary IP address of a host in Netbox.
- Initiate SSH connection to the host.
- Support for both devices and virtual machines.
- Option to specify an SSH username with a flag.
- Configuration via environment variables.

## Installation

**Download and build the source code:**

   ```sh
   git clone git@github.com:Charlie-Root/nbssh.git
   cd nbssh
   go build -o nbssh
   ```

**Use a binary**
Simply grab the binary from the releases and copy it to /bin (make sure it has the correct permissions)

## **Set environment variables:**

Ensure you have the following environment variables set, for example in  /etc/environment

   ```sh
   NETBOX_URL=https://your-netbox-instance
   NETBOX_API_TOKEN=your-netbox-api-token
  ```

## Usage
`nbssh [-u username] <hostname>
`

## Contributing
Contributions are welcome! Please fork the repository and submit pull requests.

