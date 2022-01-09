# kafka-fwd
Standalone message forwarding tool for Apache Kafka.

Kafka-fwd can:
- Continuously forwards messages from source to destination topics
- Optionally ensure processing delay based on the original message timestamp


# Download and Install
- [Download](https://github.com/pagrus7/kafka-fwd/releases) Linux or macOS (darwin) distribution
- Unpack and modify sample `kfwd-conf.yaml` to your liking
- Run `kafka-fwd`

# Build from Sources
- Clone the repo
- Run `go build` with go 1.18 or later


# License
[MIT License](https://mit-license.org/)
