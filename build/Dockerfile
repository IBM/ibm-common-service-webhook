FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/ibm-common-service-webhook \
    USER_UID=1001 \
    USER_NAME=ibm-common-service-webhook

# install operator binary
COPY build/_output/bin/ibm-cs-webhook ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
