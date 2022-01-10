FROM ubuntu:20.04

LABEL org.label-schema.schema-version="1.0"
LABEL org.label-schema.name="kafka-fwd"
LABEL org.label-schema.description="kafka-fwd image"
LABEL org.label-schema.vcs-url="https://github.com/pagrus7/kafka-fwd"

WORKDIR /opt/kafka-fwd

COPY kafka-fwd .

CMD [ "./kafka-fwd" ]