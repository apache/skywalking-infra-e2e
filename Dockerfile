# Licensed to Apache Software Foundation (ASF) under one or more contributor
# license agreements. See the NOTICE file distributed with
# this work for additional information regarding copyright
# ownership. Apache Software Foundation (ASF) licenses this file to you under
# the Apache License, Version 2.0 (the "License"); you may
# not use this file except in compliance with the License.
# You may obtain a copy of the License at
# 
#     http://www.apache.org/licenses/LICENSE-2.0
# 
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied.  See the License for the
# specific language governing permissions and limitations
# under the License.
#

FROM golang:1.16 AS build

WORKDIR /e2e

COPY . .

RUN make linux

FROM golang:1.16 AS bin

RUN apt update; \
    apt install -y docker-compose

COPY --from=build /e2e/bin/linux/e2e /usr/local/bin/e2e

# Add common tools, copy from prebuilt Docker image whenever possible.
COPY --from=stedolan/jq /usr/local/bin/jq /usr/local/bin/jq
COPY --from=mikefarah/yq:4 /usr/bin/yq /usr/local/bin/yq
COPY --from=docker /usr/local/bin/docker /usr/local/bin/docker

WORKDIR /github/workspace/

ENTRYPOINT ["e2e"]
