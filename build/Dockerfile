# Copyright 2019 HAProxy Technologies LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM haproxytech/haproxy-alpine:3.1

ARG TARGETPLATFORM

ENV S6_USER=haproxy
ENV S6_GROUP=haproxy

#COPY /fs /

RUN apk --no-cache add socat openssl util-linux htop tzdata curl libcap && \
    rm -f /usr/local/bin/dataplaneapi /usr/bin/dataplaneapi /etc/haproxy/dataplaneapi.yml && \
    setcap 'cap_net_bind_service=+ep' /usr/local/sbin/haproxy && \
    mkdir -p /usr/local/lib/haproxy/kubernetes-controller/{aux,maps,modules,failed}

COPY kubernetes-controller  ./kubernetes-controller

ENTRYPOINT ["./kubernetes-controller"]
CMD []
