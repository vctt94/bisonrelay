import 'package:bruig/components/plugins/pong_plugin.dart';
import 'package:bruig/grpc/grpc_plugin_client.dart';
import 'package:bruig/models/plugin.dart';

class PluginRegistry {
  static GamePlugin? getPlugin(String name, GrpcPluginClient grpcClient) {
    switch (name) {
      case 'pong':
        return PongGamePlugin(grpcClient);
      // case 'Rock-Paper-Scissors':
      //   return RockPaperScissorsPlugin(grpcClient); // Add more plugins as needed
      default:
        return null;
    }
  }
}
