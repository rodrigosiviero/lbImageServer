# Launchbox ImageServer

## Overview

Launchbox ImageServer is a simple web server designed to serve images from a specified folder. It can be run as a standalone application or as a Windows service. The server is built using Go and can be easily deployed using Docker.

## Features

- Serve images from a specified folder over HTTP
- Configurable via a JSON file
- Can be run as a Windows service
- Docker support for containerized deployment

## Configuration

The server configuration is stored in a `config.json` file located in the same directory as the executable. The configuration file should have the following structure:

```json
{
  "port": "8089",
  "folder": "C:/Users/<user>/LaunchBox/Images"
}
```

* port: The port on which the server will listen.
* folder: The folder from which images will be served.

## Running the Server

### Standalone Mode

To run the server in standalone mode, use the following command:

```shell
go run main.go
```

### Windows Service
To install, remove, start, stop, restart, enable, disable, or debug the service, use the manage_service.bat script with the appropriate command.

#### manage_service.bat

This batch script provides various commands to manage a service. Below are the available commands:

- `install`: Installs the service.
- `remove`: Removes the service.
- `start`: Starts the service.
- `stop`: Stops the service.
- `restart`: Restarts the service.
- `status`: Displays the current status of the service.
- `enable`: Enables the service to start automatically.
- `disable`: Disables the service from starting automatically.
- `debug`: Runs the service in debug mode.
- `config PORT FOLDER`: Configures the service with the specified port and folder.`

Example Usage

```shell
manage_service.bat install
manage_service.bat remove
manage_service.bat start
manage_service.bat stop
manage_service.bat restart
manage_service.bat status
manage_service.bat enable
manage_service.bat disable
manage_service.bat debug
manage_service.bat config 8089 C:/Launchbox
```

#### In order to install and enable service

Instead of the config command you can edit the config.json file that is on the directory as well, then just skip the config.

```
manage_service.bat install
manage_service.bat config 8089 C:/Users/<user>/LaunchBox/Images 
manage_service.bat enable
```

#### In order to delete

```
manage_service.bat stop
manage_service.bat remove
```

### Docker
To build and run the server using Docker, use the provided Dockerfile and docker-compose.yml files.

#### Build the Docker Image

```shell
docker build -t .
```

#### Run the Docker Container

Change the port if needed and change the volume with your path to the Images

```Docker
services:
  image-server:
    container_name: goserver
    image: ghcr.io/rodrigosiviero/lbimageserver:latest
    ports:
      - "8089:8089" # You can change the port here like this "<your-port>:8089"
    volumes:
      - "C:/Users/<user>/LaunchBox/Images:/images" # Windows-style absolute path
    environment:
      - PORT=8089
      - IMAGE_FOLDER=/images
    restart: always
```

Run the command:

```
docker compose up
```


The server will be accessible at http://localhost:your-port


## Development

Prerequisites

* Go 1.18 or later
* Docker

### Building the Project

To build the project, run the following command:

```go build -o image_server main.go```