import 'dart:io';
import 'package:bruig/grpc/generated/pluginrpc.pbgrpc.dart';
import 'package:grpc/grpc.dart';

class GrpcPluginClient {
  late ClientChannel _channel;
  late PluginServiceClient _stub;

  // Constructor now accepts tlsCertPath
  GrpcPluginClient(String serverAddress, int port, {String? tlsCertPath}) {
    // Set up credentials based on whether TLS is being used
    final credentials = tlsCertPath != null && tlsCertPath.isNotEmpty
        ? _createSecureCredentials(tlsCertPath)
        : const ChannelCredentials.insecure();

    // Initialize the gRPC channel and client stub
    _channel = ClientChannel(
      serverAddress,
      port: port,
      options: ChannelOptions(
        credentials: credentials,
      ),
    );
    _stub = PluginServiceClient(_channel);
  }

  // Create secure credentials using the TLS certificate
  ChannelCredentials _createSecureCredentials(String tlsCertPath) {
    final cert = File(tlsCertPath).readAsBytesSync();
    return ChannelCredentials.secure(certificates: cert);
  }

  // Call Init on the PluginService and listen to the stream
  Stream<PluginStartStreamResponse> init(
      String clientId, String pluginName) async* {
    final request = PluginStartStreamRequest()..clientId = clientId;

    try {
      final responseStream = _stub.init(request);
      await for (var response in responseStream) {
        yield response; // Yield each response back to the caller
      }
    } catch (e) {
      print('Error during Init: $e');
      rethrow;
    }
  }

  // Call Action on the PluginService
  Stream<PluginCallActionStreamResponse> callAction(
      String user, String action, List<int>? data) async* {
    final request = PluginCallActionStreamRequest()
      ..action = action
      ..user = user;

    if (data != null) {
      request.data = data;
    }

    try {
      final responseStream = _stub.callAction(request);
      await for (var response in responseStream) {
        yield response;
      }
    } catch (e) {
      print('Error during CallAction: $e');
      rethrow;
    }
  }

  // In GrpcPluginClient
  Future<void> sendInput(List<int> inputData, String user) async {
    // Implement the gRPC call to send input without expecting a stream response
    final request = PluginInputRequest()
      ..data = inputData
      ..user = user;

    await _stub.sendInput(request);
  }

  // GetVersion method (unary call)
  Future<PluginVersionResponse> getVersion() async {
    final request = PluginVersionRequest();

    try {
      final response = await _stub.getVersion(request);
      print(response);
      return response; // Return the response
    } catch (e) {
      print('Error during GetVersion: $e');
      rethrow;
    }
  }

  // Optionally, clean up the gRPC connection
  Future<void> shutdown() async {
    await _channel.shutdown();
  }
}
