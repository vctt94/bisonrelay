import 'dart:convert';

import 'package:bruig/components/plugins/plugin_registry.dart';
import 'package:bruig/grpc/grpc_plugin_client.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

// Import models
import 'package:bruig/models/client.dart';
import 'package:bruig/models/plugin.dart';
import 'package:bruig/models/snackbar.dart';

class PluginContentScreenArgs {
  final PluginModel plugin;
  final GrpcPluginClient grpcClient;

  PluginContentScreenArgs(this.plugin, this.grpcClient);
}

class PluginContentScreen extends StatelessWidget {
  final PluginContentScreenArgs args;
  final Function tabChange;
  const PluginContentScreen(this.args, this.tabChange, {Key? key})
      : super(key: key);

  @override
  Widget build(BuildContext context) {
    return ChangeNotifierProvider<PluginModel>.value(
      value: args.plugin,
      child: Consumer<ClientModel>(
        builder: (context, client, child) =>
            _PluginContentScreenForArgs(args, client, tabChange),
      ),
    );
  }
}

class _PluginContentScreenForArgs extends StatefulWidget {
  final PluginContentScreenArgs args;
  final ClientModel client;
  final Function tabChange;

  const _PluginContentScreenForArgs(this.args, this.client, this.tabChange,
      {Key? key})
      : super(key: key);

  @override
  State<_PluginContentScreenForArgs> createState() =>
      _PluginContentScreenForArgsState();
}

class _PluginContentScreenForArgsState
    extends State<_PluginContentScreenForArgs> {
  bool loading = false;
  String pluginData = "";
  final FocusNode _focusNode = FocusNode();
  TextEditingController inputController = TextEditingController();
  GamePlugin? plugin; // Plugin object

  @override
  void initState() {
    super.initState();
    plugin = PluginRegistry.getPlugin(
        widget.args.plugin.name, widget.args.grpcClient);
  }

  @override
  void dispose() {
    _focusNode.dispose();
    inputController.dispose();
    super.dispose();
  }

  void _handleRawKeyEvent(RawKeyEvent event) {
    if (event is RawKeyDownEvent) {
      LogicalKeyboardKey key = event.logicalKey;

      print(key.keyLabel);
      plugin?.handleInput(
        context,
        widget.client.publicID,
        key.keyLabel,
      );
    }
  }

  void _handlePaddleMovement(DragUpdateDetails details) {
    double deltaY = details.delta.dy;
    String data = deltaY < 0 ? 'ArrowUp' : 'ArrowDown';
    plugin?.handleInput(context, widget.client.publicID, data);
  }

  void sendAction(String input) {
    if (input.isNotEmpty) {
      String action;
      List<int>? data;

      action = input.trim();
      print(action);

      // Call performAction with the action and byte array (or null)
      widget.args.plugin.performAction(
        context,
        widget.client.publicID,
        action,
        data,
      );
      inputController.clear();
    }
  }

  @override
  Widget build(BuildContext context) {
    if (loading) {
      return const Center(child: CircularProgressIndicator());
    }

    if (plugin == null) {
      return const Center(child: Text('Loading plugin...'));
    }

    // Wrap the widget where the game state changes need to be reflected inside a `Consumer`
    return Scaffold(
      appBar: AppBar(
        title: Text(plugin!.name),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () => Navigator.pop(context),
        ),
      ),
      body: RawKeyboardListener(
        focusNode: _focusNode,
        autofocus: false,
        onKey: _handleRawKeyEvent,
        child: Consumer<PluginModel>(
          builder: (context, pluginModel, _) {
            return Column(
              children: [
                // Plugin Header
                Container(
                  padding: const EdgeInsets.all(16),
                  color: Colors.blueGrey[100],
                  child: Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      Text(
                        widget.args.plugin.name,
                        style: const TextStyle(
                            fontWeight: FontWeight.bold, fontSize: 20),
                      ),
                      Text(
                        widget.args.plugin.installationDate
                            .toLocal()
                            .toIso8601String(),
                      ),
                    ],
                  ),
                ),
                // Dynamically load the correct plugin widget
                Expanded(
                  child: plugin!.buildWidget(
                    pluginModel.gameState, // Pass updated gameState
                    _focusNode,
                    _handlePaddleMovement,
                  ),
                ),
                // Input and controls section
                Container(
                  padding: const EdgeInsets.all(16),
                  child: Row(
                    children: [
                      Expanded(
                        child: TextField(
                          controller: inputController,
                          decoration: const InputDecoration(
                            labelText: "Enter action",
                            border: OutlineInputBorder(),
                          ),
                        ),
                      ),
                      const SizedBox(width: 10),
                      ElevatedButton(
                        onPressed: () {
                          if (inputController.text.isNotEmpty) {
                            sendAction(inputController.text);
                          }
                        },
                        child: const Text("Send"),
                      ),
                    ],
                  ),
                ),
              ],
            );
          },
        ),
      ),
    );
  }
}
