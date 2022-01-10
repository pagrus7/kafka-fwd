# kafka-fwd

Standalone message forwarding tool for Apache Kafka. Build in Go,
leverages Confluent [client for Apache Kafka](https://github.com/confluentinc/confluent-kafka-go)

Major use case behind kafka-fwd is [failed] message reprocessing.

### Features

- Forward messages from src to dest topic
  - Forward continuously
  - Support multiple routes
- Ensure fixed delay before forwarding
  - E.g. forward if 1min 30sec elapsed since original message timestamp
- Respect source messages
  - Don't skip, at the cost of potential duplicates and out-of-order delivery
- Provide reasonable performance and scalability
  - Batching, consumer groups, configurability: don't stand in the way of Confluent client & [libkafkard](https://github.com/edenhill/librdkafka)
  - Scalable delay: limit in-flight messages and pause/resume partitions as needed


# Download and Install

- [Download](https://github.com/pagrus7/kafka-fwd/releases) Linux or macOS (darwin) distribution
- Unpack and modify sample `kfwd-conf.yaml` to your liking
- Run `kafka-fwd`

### External Restart

kafka-fwd may panic and terminate if unable to guarantee a lossless delivery of forwarded messages. It is safe to
restart then. Currently, kafka-fwd expects to be restarted externally, e.g. with docker restart policy.

# Build from Sources

- Clone the repo
- Run `go build` with go 1.18 or later

# License

[MIT License](https://mit-license.org/)
