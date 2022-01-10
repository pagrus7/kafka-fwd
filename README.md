# kafka-fwd
Standalone message forwarding tool for Apache Kafka.

Kafka-fwd can:
- Continuously forward messages from source to destination topics
- Optionally, ensure fixed delay since the original message timestamp


# Download and Install
- [Download](https://github.com/pagrus7/kafka-fwd/releases) Linux or macOS (darwin) distribution
- Unpack and modify sample `kfwd-conf.yaml` to your liking
- Run `kafka-fwd`

### External Restart
kafka-fwd may panic and terminate if unable to guarantee a lossless delivery of forwarded messages.
It is safe to restart then. Currently, kafka-fwd expects to be restarted externally, e.g. with docker restart policy.   

# Build from Sources
- Clone the repo
- Run `go build` with go 1.18 or later


# License
[MIT License](https://mit-license.org/)
