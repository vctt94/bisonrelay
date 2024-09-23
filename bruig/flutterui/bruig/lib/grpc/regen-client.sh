#!/bin/sh

# Function to install the Dart protoc plugin
build_protoc_gen_dart() {
    dart pub global activate protoc_plugin
}

# Function to generate Dart gRPC code from .proto files
generate() {
    protoc --dart_out=grpc:./generated -I../../../../../clientplugin ../../../../../clientplugin/pluginrpc.proto
}

# Build the Dart protoc plugin (if not installed already)
build_protoc_gen_dart

# Run the generate function
generate
