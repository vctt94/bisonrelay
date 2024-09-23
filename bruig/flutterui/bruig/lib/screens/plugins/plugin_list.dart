import 'package:bruig/components/empty_widget.dart';
import 'package:bruig/components/text.dart';
import 'package:bruig/grpc/grpc_plugin_client.dart';
import 'package:bruig/models/client.dart';
import 'package:bruig/models/plugin.dart';
import 'package:bruig/models/snackbar.dart';
import 'package:bruig/models/uistate.dart';
import 'package:bruig/screens/ln/components.dart';
import 'package:bruig/screens/plugins/plugin_content.dart';
import 'package:flutter/material.dart';
import 'package:golib_plugin/golib_plugin.dart';
import 'package:provider/provider.dart';
import 'package:bruig/theme_manager.dart';

class PluginListScreen extends StatefulWidget {
  static String routeName = "/pluginList";
  final ClientModel client;
  const PluginListScreen(this.client, {Key? key}) : super(key: key);

  @override
  State<PluginListScreen> createState() => _PluginListScreenState();
}

typedef _UninstallFunc = void Function(String pluginId);

// Method to split the address into host and port
List<String> extractHostAndPort(String addressWithPort) {
  final parts = addressWithPort.split(':');
  if (parts.length == 2) {
    return [parts[0], parts[1]]; // [host, port]
  }
  return [addressWithPort, '50051']; // Default port
}

class _PluginItem extends StatelessWidget {
  final int index;
  final String pluginId;
  final String pluginName;
  final Map<String, dynamic> config;
  final bool isInstalled;
  final _UninstallFunc uninstall;
  final ClientModel client;

  const _PluginItem(this.index, this.pluginId, this.pluginName, this.config,
      this.isInstalled, this.uninstall, this.client,
      {Key? key})
      : super(key: key);

  @override
  Widget build(BuildContext context) {
    bool isScreenSmall = checkIsScreenSmall(context);

    return GestureDetector(
      onTap: () {
        // Create PluginModel instance
        PluginModel pluginModel = PluginModel();

        // Extract the TLSCertPath and Address from config
        String tlsCertPath = config['TLSCertPath'] ?? "";
        String address = config['Address'] ?? "";

        final hostAndPort = extractHostAndPort(address);
        final host = hostAndPort[0];
        final port = int.parse(hostAndPort[1]);
        // Create GrpcPluginClient with the extracted details
        GrpcPluginClient grpcClient =
            GrpcPluginClient(host, port, tlsCertPath: tlsCertPath);

        // Initialize the plugin with client model, clientId (application ID), pluginName, certPath, and address
        pluginModel.initialize(
            context, client, grpcClient, pluginId, pluginName);

        // Navigate to the PluginContentScreen with the plugin model and GrpcPluginClient
        Navigator.push(
          context,
          MaterialPageRoute(
            builder: (context) => PluginContentScreen(
              PluginContentScreenArgs(pluginModel, grpcClient),
              (tabIndex) {
                // Handle tab change if necessary
              },
            ),
          ),
        );
      },
      child: Consumer<ThemeNotifier>(
        builder: (context, theme, _) => Container(
          color: index.isEven
              ? theme.colors.surfaceContainerHigh
              : theme.colors.surface,
          margin: isScreenSmall
              ? const EdgeInsets.only(left: 10, right: 10, top: 8)
              : const EdgeInsets.only(left: 50, right: 50, top: 8),
          padding: const EdgeInsets.only(top: 4, bottom: 4, left: 8, right: 8),
          child: Row(
            children: [
              Expanded(
                flex: 8,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Txt.S("Name: $pluginName", overflow: TextOverflow.ellipsis),
                    Txt.S("ID: $pluginId", overflow: TextOverflow.ellipsis),
                  ],
                ),
              ),
              (isInstalled
                  ? IconButton(
                      visualDensity: VisualDensity.compact,
                      tooltip: "Uninstall Plugin",
                      onPressed: () {
                        uninstall(pluginId);
                      },
                      icon: const Icon(Icons.remove_circle_outline_rounded))
                  : const Empty())
            ],
          ),
        ),
      ),
    );
  }
}

class _PluginListScreenState extends State<PluginListScreen> {
  bool firstLoading = true;
  ScrollController installedPluginsCtrl = ScrollController();
  ScrollController availablePluginsCtrl = ScrollController();
  List<Map<String, dynamic>> installedPlugins = [];
  List<Map<String, dynamic>> availablePlugins = [];
  ClientModel get client => widget.client;

  void loadPlugins() async {
    var snackbar = SnackBarModel.of(context);
    try {
      var newInstalledPlugins = await Golib.listInstalledPlugins();
      var newAvailablePlugins = await Golib.listAvailablePlugins();

      setState(() {
        installedPlugins = newInstalledPlugins;
        availablePlugins = newAvailablePlugins;
      });
    } catch (exception) {
      snackbar.error("Unable to load plugin lists: $exception");
    } finally {
      setState(() {
        firstLoading = false;
      });
    }
  }

  void uninstallPlugin(String pluginId) async {
    // await Golib.uninstallPlugin(pluginId);
    loadPlugins();
  }

  @override
  void initState() {
    super.initState();
    loadPlugins();
  }

  @override
  Widget build(BuildContext context) {
    if (firstLoading) {
      return const Text("Loading...");
    }

    return Consumer<ThemeNotifier>(
        builder: (context, theme, _) => Container(
            padding: const EdgeInsets.all(16),
            child: Column(children: [
              const LNInfoSectionHeader("Installed Plugins"),
              const SizedBox(height: 20),
              Expanded(
                  child: ListView.builder(
                      controller: installedPluginsCtrl,
                      itemCount: installedPlugins.length,
                      itemBuilder: (context, index) => _PluginItem(
                          index,
                          installedPlugins[index]['id'], // Passing plugin ID
                          installedPlugins[index]
                              ['name'], // Passing plugin name
                          installedPlugins[index]
                              ['config'], // Passing plugin config
                          true,
                          uninstallPlugin,
                          client // Pass the ClientModel to PluginItem
                          ))),
              const LNInfoSectionHeader("Available Plugins"),
              const SizedBox(height: 20),
              Expanded(
                  child: ListView.builder(
                controller: availablePluginsCtrl,
                itemCount: availablePlugins.length,
                itemBuilder: (context, index) => _PluginItem(
                    index,
                    availablePlugins[index]['id'], // Passing plugin ID
                    availablePlugins[index]['name'], // Passing plugin name
                    availablePlugins[index]['config'], // Passing plugin config
                    false,
                    uninstallPlugin,
                    client),
              )),
            ])));
  }
}
