import 'dart:async';
import 'dart:convert';

import 'package:flutter/cupertino.dart';

import 'package:bruig/grpc/generated/pluginrpc.pb.dart';
import 'package:bruig/grpc/grpc_plugin_client.dart';
import 'package:bruig/models/client.dart';
import 'package:bruig/models/snackbar.dart';

enum PluginState { idle, ready, playing }

abstract class GamePlugin {
  Widget buildWidget(Map<String, dynamic> gameState, FocusNode focusNode,
      Function(DragUpdateDetails) handlePaddleMovement);
  void handleInput(BuildContext context, String clientId, String data);
  String get name;
}

class PluginModel extends ChangeNotifier {
  PluginState _state = PluginState.idle;
  StreamSubscription<PluginCallActionStreamResponse>? _actionSubscription;

  Map<String, dynamic> gameState = {}; // Store the game state

  PluginState get state => _state;

  void setState(PluginState newState) {
    _state = newState;
    notifyListeners();
  }

  String id = "";
  String name = "";
  String details = "";

  DateTime _installationDate = DateTime.now();
  DateTime get installationDate => _installationDate;
  set installationDate(DateTime ts) {
    _installationDate = ts;
    notifyListeners();
  }

  bool _active = false;
  bool get active => _active;
  set active(bool b) {
    _active = b;
    notifyListeners();
  }

  // Initialize the gRPC plugin client
  GrpcPluginClient? grpcClient;

  PluginModel();

  Future<void> initialize(
    BuildContext context,
    ClientModel client,
    GrpcPluginClient grpcClient,
    String pluginID,
    String name,
  ) async {
    this.grpcClient = grpcClient;
    this.id = pluginID;
    this.name = name;

    // Extract the host and port from the address

    // Initialize the gRPC client with the given address (hostname) and certificate

    notifyListeners();

    try {
      // Call the Init gRPC method to initialize the plugin
      var responseStream = grpcClient.init(client.publicID, name);

      // Start listening to the initialization stream and handle responses
      _startNotificationStream(responseStream, context);
    } catch (e) {
      print("Error initializing plugin: $e");
      SnackBarModel.of(context).error('Error initializing plugin: $e');
      throw e;
    }
  }

  // Perform the action, calling gRPC
  Future<void> performAction(
    BuildContext context,
    String user,
    String action,
    List<int>? data,
  ) async {
    // Cancel any existing subscription
    await _actionSubscription?.cancel();

    try {
      // Start the game by calling the gRPC method for the action
      var responseStream = grpcClient!.callAction(user, action, data);

      print("Action: $action");
      // Listen to the game updates
      _actionSubscription = responseStream.listen((response) {
        final String decodedResponse =
            utf8.decode(response.response); // Decode the byte response

        // Parse the response as JSON for game state or other purposes
        try {
          gameState = jsonDecode(decodedResponse);
          notifyListeners(); // Notify listeners that the game state has been updated
        } catch (e) {
          print('Error parsing game state: $decodedResponse');
        }
      });
    } catch (e) {
      print('Failed to perform action: $e');
    }
  }

  // Listen to the gRPC notification stream
  void _startNotificationStream(
      Stream<PluginStartStreamResponse> responseStream,
      BuildContext context) async {
    await for (var response in responseStream) {
      handleInitResponse(response, context);
    }
  }

  // Reset the state when the game ends
  void reset() {
    _state = PluginState.idle;
    gameState.clear();
    notifyListeners();
  }

  // Handle the Init stream responses from gRPC
  void handleInitResponse(
      PluginStartStreamResponse response, BuildContext context) {
    print("Plugin initialized with response: ${response.message}");

    if (response.started) {
      setState(PluginState.playing); // Move to playing state
    }

    // Notify listeners after updating the state
    notifyListeners();

    // Show a success message using SnackBarModel
    SnackBarModel.of(context)
        .success("Notification from plugin: ${response.message}");
  }

  Future<void> loadPluginDetails() async {
    try {
      details = "This is the detailed description of the plugin: $name.";
      notifyListeners();
    } catch (e) {
      throw "Error loading plugin details: $e";
    }
  }

  Future<void> activatePlugin() async {
    _active = true;
    notifyListeners();
  }

  Future<void> deactivatePlugin() async {
    _active = false;
    notifyListeners();
  }

  // Clean up the gRPC client
  Future<void> shutdownGrpcClient() async {
    if (grpcClient != null) {
      await grpcClient!.shutdown();
    }
  }
}
