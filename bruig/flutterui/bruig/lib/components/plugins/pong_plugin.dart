import 'dart:convert';

import 'package:bruig/grpc/grpc_plugin_client.dart';
import 'package:bruig/models/plugin.dart';
import 'package:bruig/models/snackbar.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter/rendering.dart';

class PongGamePlugin implements GamePlugin {
  final GrpcPluginClient grpcClient; // gRPC client instance

  PongGamePlugin(this.grpcClient);

  @override
  Widget buildWidget(Map<String, dynamic> gameState, FocusNode focusNode,
      Function(DragUpdateDetails) handlePaddleMovement) {
    return GestureDetector(
      onPanUpdate: handlePaddleMovement,
      onTap: () => focusNode.requestFocus(),
      child: LayoutBuilder(
        builder: (context, constraints) {
          return CustomPaint(
            size: Size(constraints.maxWidth, constraints.maxHeight),
            painter: PongPainter(gameState),
          );
        },
      ),
    );
  }

  @override
  @override
  Future<void> handleInput(
      BuildContext context, String clientId, String data) async {
    try {
      // Decode the raw input data to get the key label

      String action;

      // Translate raw key label to action
      if (data == 'W' || data == 'ArrowUp') {
        action = 'up';
      } else if (data == 'S' || data == 'ArrowDown') {
        action = 'down';
      } else {
        // Ignore unhandled keys
        return;
      }

      // Encode the action and send it via gRPC
      List<int> encodedAction = utf8.encode(action);
      await grpcClient.sendInput(encodedAction, clientId);
    } catch (e) {
      SnackBarModel.of(context).error('Failed to send input via gRPC: $e');
    }
  }

  @override
  String get name => 'Pong';
}

class PongPainter extends CustomPainter {
  final Map<String, dynamic> gameState;

  PongPainter(this.gameState);

  @override
  void paint(Canvas canvas, Size size) {
    // Extract game dimensions
    double gameWidth = (gameState['gameWidth'] as num?)?.toDouble() ?? 80.0;
    double gameHeight = (gameState['gameHeight'] as num?)?.toDouble() ?? 40.0;

    // Calculate scaling factors
    double scaleX = size.width / gameWidth;
    double scaleY = size.height / gameHeight;

    // Extract and scale paddle 1 properties
    double paddle1X = 0.0; // Paddle 1 is on the left edge
    double paddle1Y = (gameState['p1Y'] as num?)?.toDouble() ?? 0.0;
    double paddle1Width = (gameState['p1Width'] as num?)?.toDouble() ?? 1.0;
    double paddle1Height = (gameState['p1Height'] as num?)?.toDouble() ?? 5.0;

    // Scale paddle 1 properties
    paddle1X *= scaleX;
    paddle1Y *= scaleY;
    paddle1Width *= scaleX;
    paddle1Height *= scaleY;

    // Extract and scale paddle 2 properties
    double paddle2X = (gameState['p2X'] as num?)?.toDouble() ?? gameWidth - 1.0;
    double paddle2Y = (gameState['p2Y'] as num?)?.toDouble() ?? 0.0;
    double paddle2Width = (gameState['p2Width'] as num?)?.toDouble() ?? 1.0;
    double paddle2Height = (gameState['p2Height'] as num?)?.toDouble() ?? 5.0;

    // Scale paddle 2 properties
    paddle2X *= scaleX;
    paddle2Y *= scaleY;
    paddle2Width *= scaleX;
    paddle2Height *= scaleY;

    // Extract and scale ball properties
    double ballX = (gameState['ballX'] as num?)?.toDouble() ?? gameWidth / 2;
    double ballY = (gameState['ballY'] as num?)?.toDouble() ?? gameHeight / 2;
    double ballWidth = (gameState['ballWidth'] as num?)?.toDouble() ?? 3.0;
    double ballHeight = (gameState['ballHeight'] as num?)?.toDouble() ?? 3.0;

    // Scale ball properties
    ballX *= scaleX;
    ballY *= scaleY;
    ballWidth *= scaleX;
    ballHeight *= scaleY;

    // Paint object for drawing
    var paint = Paint()
      ..color = Colors.white
      ..style = PaintingStyle.fill;

    // Draw background
    canvas.drawRect(
      Rect.fromLTWH(0.0, 0.0, size.width, size.height),
      Paint()..color = Colors.black,
    );

    // Draw Paddle 1
    canvas.drawRect(
      Rect.fromLTWH(paddle1X, paddle1Y, paddle1Width, paddle1Height),
      paint,
    );

    // Draw Paddle 2
    canvas.drawRect(
      Rect.fromLTWH(paddle2X, paddle2Y, paddle2Width, paddle2Height),
      paint,
    );

    // Draw the ball
    canvas.drawRect(
      Rect.fromLTWH(ballX, ballY, ballWidth, ballHeight),
      paint,
    );
  }

  @override
  bool shouldRepaint(PongPainter oldDelegate) {
    // Repaint whenever the game state changes
    return oldDelegate.gameState != gameState;
  }
}
