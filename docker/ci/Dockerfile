# stage 1: building application binary file
FROM golang:1.19.9-alpine3.18 as build

RUN mkdir /source
COPY . /source
WORKDIR /source

ARG COMMIT
RUN go mod download
RUN CGO_ENABLED=0 go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X main.commit=${COMMIT}" -o main .


# stage 2: copy only the application binary file and necessary files to the alpine container
FROM alpine:3.12 as run

RUN apk --no-cache add ca-certificates

ARG USER=f764a076-ae85-4b9a-b258-6998694ee8a6
ENV HOME /home/$USER

RUN mkdir $HOME
RUN mkdir $HOME/ssl
RUN mkdir $HOME/data
RUN mkdir $HOME/logs

# add new user
RUN apk add --update sudo
RUN adduser -D $USER \
    && mkdir -p /etc/sudoers.d \
    && echo "$USER ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/$USER \
    && chmod 0440 /etc/sudoers.d/$USER

RUN sudo chown -R $USER:$USER $HOME
RUN sudo chmod 777 $HOME

WORKDIR /home/$USER

COPY --from=build /source/main .
COPY --from=build /source/database ./database

USER $USER

EXPOSE 5003

ENTRYPOINT ["./main"]