//
//  Generated code. Do not modify.
//  source: pluginrpc.proto
//
// @dart = 2.12

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_final_fields
// ignore_for_file: unnecessary_import, unnecessary_this, unused_import

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:grpc/service_api.dart' as $grpc;
import 'package:protobuf/protobuf.dart' as $pb;

import 'pluginrpc.pb.dart' as $0;

export 'pluginrpc.pb.dart';

@$pb.GrpcServiceName('PluginService')
class PluginServiceClient extends $grpc.Client {
  static final _$init = $grpc.ClientMethod<$0.PluginStartStreamRequest, $0.PluginStartStreamResponse>(
      '/PluginService/Init',
      ($0.PluginStartStreamRequest value) => value.writeToBuffer(),
      ($core.List<$core.int> value) => $0.PluginStartStreamResponse.fromBuffer(value));
  static final _$callAction = $grpc.ClientMethod<$0.PluginCallActionStreamRequest, $0.PluginCallActionStreamResponse>(
      '/PluginService/CallAction',
      ($0.PluginCallActionStreamRequest value) => value.writeToBuffer(),
      ($core.List<$core.int> value) => $0.PluginCallActionStreamResponse.fromBuffer(value));
  static final _$sendInput = $grpc.ClientMethod<$0.PluginInputRequest, $0.PluginInputResponse>(
      '/PluginService/SendInput',
      ($0.PluginInputRequest value) => value.writeToBuffer(),
      ($core.List<$core.int> value) => $0.PluginInputResponse.fromBuffer(value));
  static final _$getVersion = $grpc.ClientMethod<$0.PluginVersionRequest, $0.PluginVersionResponse>(
      '/PluginService/GetVersion',
      ($0.PluginVersionRequest value) => value.writeToBuffer(),
      ($core.List<$core.int> value) => $0.PluginVersionResponse.fromBuffer(value));
  static final _$render = $grpc.ClientMethod<$0.RenderRequest, $0.RenderResponse>(
      '/PluginService/Render',
      ($0.RenderRequest value) => value.writeToBuffer(),
      ($core.List<$core.int> value) => $0.RenderResponse.fromBuffer(value));

  PluginServiceClient($grpc.ClientChannel channel,
      {$grpc.CallOptions? options,
      $core.Iterable<$grpc.ClientInterceptor>? interceptors})
      : super(channel, options: options,
        interceptors: interceptors);

  $grpc.ResponseStream<$0.PluginStartStreamResponse> init($0.PluginStartStreamRequest request, {$grpc.CallOptions? options}) {
    return $createStreamingCall(_$init, $async.Stream.fromIterable([request]), options: options);
  }

  $grpc.ResponseStream<$0.PluginCallActionStreamResponse> callAction($0.PluginCallActionStreamRequest request, {$grpc.CallOptions? options}) {
    return $createStreamingCall(_$callAction, $async.Stream.fromIterable([request]), options: options);
  }

  $grpc.ResponseFuture<$0.PluginInputResponse> sendInput($0.PluginInputRequest request, {$grpc.CallOptions? options}) {
    return $createUnaryCall(_$sendInput, request, options: options);
  }

  $grpc.ResponseFuture<$0.PluginVersionResponse> getVersion($0.PluginVersionRequest request, {$grpc.CallOptions? options}) {
    return $createUnaryCall(_$getVersion, request, options: options);
  }

  $grpc.ResponseFuture<$0.RenderResponse> render($0.RenderRequest request, {$grpc.CallOptions? options}) {
    return $createUnaryCall(_$render, request, options: options);
  }
}

@$pb.GrpcServiceName('PluginService')
abstract class PluginServiceBase extends $grpc.Service {
  $core.String get $name => 'PluginService';

  PluginServiceBase() {
    $addMethod($grpc.ServiceMethod<$0.PluginStartStreamRequest, $0.PluginStartStreamResponse>(
        'Init',
        init_Pre,
        false,
        true,
        ($core.List<$core.int> value) => $0.PluginStartStreamRequest.fromBuffer(value),
        ($0.PluginStartStreamResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.PluginCallActionStreamRequest, $0.PluginCallActionStreamResponse>(
        'CallAction',
        callAction_Pre,
        false,
        true,
        ($core.List<$core.int> value) => $0.PluginCallActionStreamRequest.fromBuffer(value),
        ($0.PluginCallActionStreamResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.PluginInputRequest, $0.PluginInputResponse>(
        'SendInput',
        sendInput_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.PluginInputRequest.fromBuffer(value),
        ($0.PluginInputResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.PluginVersionRequest, $0.PluginVersionResponse>(
        'GetVersion',
        getVersion_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.PluginVersionRequest.fromBuffer(value),
        ($0.PluginVersionResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.RenderRequest, $0.RenderResponse>(
        'Render',
        render_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.RenderRequest.fromBuffer(value),
        ($0.RenderResponse value) => value.writeToBuffer()));
  }

  $async.Stream<$0.PluginStartStreamResponse> init_Pre($grpc.ServiceCall call, $async.Future<$0.PluginStartStreamRequest> request) async* {
    yield* init(call, await request);
  }

  $async.Stream<$0.PluginCallActionStreamResponse> callAction_Pre($grpc.ServiceCall call, $async.Future<$0.PluginCallActionStreamRequest> request) async* {
    yield* callAction(call, await request);
  }

  $async.Future<$0.PluginInputResponse> sendInput_Pre($grpc.ServiceCall call, $async.Future<$0.PluginInputRequest> request) async {
    return sendInput(call, await request);
  }

  $async.Future<$0.PluginVersionResponse> getVersion_Pre($grpc.ServiceCall call, $async.Future<$0.PluginVersionRequest> request) async {
    return getVersion(call, await request);
  }

  $async.Future<$0.RenderResponse> render_Pre($grpc.ServiceCall call, $async.Future<$0.RenderRequest> request) async {
    return render(call, await request);
  }

  $async.Stream<$0.PluginStartStreamResponse> init($grpc.ServiceCall call, $0.PluginStartStreamRequest request);
  $async.Stream<$0.PluginCallActionStreamResponse> callAction($grpc.ServiceCall call, $0.PluginCallActionStreamRequest request);
  $async.Future<$0.PluginInputResponse> sendInput($grpc.ServiceCall call, $0.PluginInputRequest request);
  $async.Future<$0.PluginVersionResponse> getVersion($grpc.ServiceCall call, $0.PluginVersionRequest request);
  $async.Future<$0.RenderResponse> render($grpc.ServiceCall call, $0.RenderRequest request);
}
