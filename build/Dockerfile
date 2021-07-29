FROM registry.access.redhat.com/ubi8/ubi-minimal:8.4
ARG VCS_REF
ARG VCS_URL

LABEL org.label-schema.vendor="IBM" \
  org.label-schema.name="ibm common service webhook" \
  org.label-schema.description="Webhook of IBM common service" \
  org.label-schema.vcs-ref=$VCS_REF \
  org.label-schema.vcs-url=$VCS_URL \
  org.label-schema.license="Licensed Materials - Property of IBM" \
  org.label-schema.schema-version="1.0" \
  name="ibm common service webhook" \
  vendor="IBM" \
  description="Webhook of IBM common service" \
  summary="Webhook of IBM common service"

ENV OPERATOR=/usr/local/bin/ibm-common-service-webhook \
    USER_UID=1001 \
    USER_NAME=ibm-common-service-webhook

# install operator binary
COPY build/_output/bin/ibm-cs-webhook ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

# copy licenses
RUN mkdir /licenses
COPY LICENSE /licenses

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
