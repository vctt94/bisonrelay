//
//  Generated code. Do not modify.
//  source: pluginrpc.proto
//
// @dart = 2.12

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_final_fields
// ignore_for_file: unnecessary_import, unnecessary_this, unused_import

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

class RenderRequest extends $pb.GeneratedMessage {
  factory RenderRequest({
    $core.List<$core.int>? data,
  }) {
    final $result = create();
    if (data != null) {
      $result.data = data;
    }
    return $result;
  }
  RenderRequest._() : super();
  factory RenderRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory RenderRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'RenderRequest', createEmptyInstance: create)
    ..a<$core.List<$core.int>>(1, _omitFieldNames ? '' : 'data', $pb.PbFieldType.OY)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  RenderRequest clone() => RenderRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  RenderRequest copyWith(void Function(RenderRequest) updates) => super.copyWith((message) => updates(message as RenderRequest)) as RenderRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RenderRequest create() => RenderRequest._();
  RenderRequest createEmptyInstance() => create();
  static $pb.PbList<RenderRequest> createRepeated() => $pb.PbList<RenderRequest>();
  @$core.pragma('dart2js:noInline')
  static RenderRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<RenderRequest>(create);
  static RenderRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<$core.int> get data => $_getN(0);
  @$pb.TagNumber(1)
  set data($core.List<$core.int> v) { $_setBytes(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasData() => $_has(0);
  @$pb.TagNumber(1)
  void clearData() => clearField(1);
}

class RenderResponse extends $pb.GeneratedMessage {
  factory RenderResponse({
    $core.String? data,
  }) {
    final $result = create();
    if (data != null) {
      $result.data = data;
    }
    return $result;
  }
  RenderResponse._() : super();
  factory RenderResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory RenderResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'RenderResponse', createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'data')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  RenderResponse clone() => RenderResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  RenderResponse copyWith(void Function(RenderResponse) updates) => super.copyWith((message) => updates(message as RenderResponse)) as RenderResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RenderResponse create() => RenderResponse._();
  RenderResponse createEmptyInstance() => create();
  static $pb.PbList<RenderResponse> createRepeated() => $pb.PbList<RenderResponse>();
  @$core.pragma('dart2js:noInline')
  static RenderResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<RenderResponse>(create);
  static RenderResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get data => $_getSZ(0);
  @$pb.TagNumber(1)
  set data($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasData() => $_has(0);
  @$pb.TagNumber(1)
  void clearData() => clearField(1);
}

class PluginStartStreamRequest extends $pb.GeneratedMessage {
  factory PluginStartStreamRequest({
    $core.String? clientId,
  }) {
    final $result = create();
    if (clientId != null) {
      $result.clientId = clientId;
    }
    return $result;
  }
  PluginStartStreamRequest._() : super();
  factory PluginStartStreamRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory PluginStartStreamRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'PluginStartStreamRequest', createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'clientId')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  PluginStartStreamRequest clone() => PluginStartStreamRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  PluginStartStreamRequest copyWith(void Function(PluginStartStreamRequest) updates) => super.copyWith((message) => updates(message as PluginStartStreamRequest)) as PluginStartStreamRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginStartStreamRequest create() => PluginStartStreamRequest._();
  PluginStartStreamRequest createEmptyInstance() => create();
  static $pb.PbList<PluginStartStreamRequest> createRepeated() => $pb.PbList<PluginStartStreamRequest>();
  @$core.pragma('dart2js:noInline')
  static PluginStartStreamRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<PluginStartStreamRequest>(create);
  static PluginStartStreamRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get clientId => $_getSZ(0);
  @$pb.TagNumber(1)
  set clientId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasClientId() => $_has(0);
  @$pb.TagNumber(1)
  void clearClientId() => clearField(1);
}

class PluginStartStreamResponse extends $pb.GeneratedMessage {
  factory PluginStartStreamResponse({
    $core.String? clientId,
    $core.bool? started,
    $core.String? message,
  }) {
    final $result = create();
    if (clientId != null) {
      $result.clientId = clientId;
    }
    if (started != null) {
      $result.started = started;
    }
    if (message != null) {
      $result.message = message;
    }
    return $result;
  }
  PluginStartStreamResponse._() : super();
  factory PluginStartStreamResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory PluginStartStreamResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'PluginStartStreamResponse', createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'clientId')
    ..aOB(2, _omitFieldNames ? '' : 'started')
    ..aOS(3, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  PluginStartStreamResponse clone() => PluginStartStreamResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  PluginStartStreamResponse copyWith(void Function(PluginStartStreamResponse) updates) => super.copyWith((message) => updates(message as PluginStartStreamResponse)) as PluginStartStreamResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginStartStreamResponse create() => PluginStartStreamResponse._();
  PluginStartStreamResponse createEmptyInstance() => create();
  static $pb.PbList<PluginStartStreamResponse> createRepeated() => $pb.PbList<PluginStartStreamResponse>();
  @$core.pragma('dart2js:noInline')
  static PluginStartStreamResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<PluginStartStreamResponse>(create);
  static PluginStartStreamResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get clientId => $_getSZ(0);
  @$pb.TagNumber(1)
  set clientId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasClientId() => $_has(0);
  @$pb.TagNumber(1)
  void clearClientId() => clearField(1);

  @$pb.TagNumber(2)
  $core.bool get started => $_getBF(1);
  @$pb.TagNumber(2)
  set started($core.bool v) { $_setBool(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasStarted() => $_has(1);
  @$pb.TagNumber(2)
  void clearStarted() => clearField(2);

  @$pb.TagNumber(3)
  $core.String get message => $_getSZ(2);
  @$pb.TagNumber(3)
  set message($core.String v) { $_setString(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasMessage() => $_has(2);
  @$pb.TagNumber(3)
  void clearMessage() => clearField(3);
}

class PluginVersionRequest extends $pb.GeneratedMessage {
  factory PluginVersionRequest() => create();
  PluginVersionRequest._() : super();
  factory PluginVersionRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory PluginVersionRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'PluginVersionRequest', createEmptyInstance: create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  PluginVersionRequest clone() => PluginVersionRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  PluginVersionRequest copyWith(void Function(PluginVersionRequest) updates) => super.copyWith((message) => updates(message as PluginVersionRequest)) as PluginVersionRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginVersionRequest create() => PluginVersionRequest._();
  PluginVersionRequest createEmptyInstance() => create();
  static $pb.PbList<PluginVersionRequest> createRepeated() => $pb.PbList<PluginVersionRequest>();
  @$core.pragma('dart2js:noInline')
  static PluginVersionRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<PluginVersionRequest>(create);
  static PluginVersionRequest? _defaultInstance;
}

/// VersionResponse is the information about the running RPC server.
class PluginVersionResponse extends $pb.GeneratedMessage {
  factory PluginVersionResponse({
    $core.String? appVersion,
    $core.String? goRuntime,
    $core.String? appName,
  }) {
    final $result = create();
    if (appVersion != null) {
      $result.appVersion = appVersion;
    }
    if (goRuntime != null) {
      $result.goRuntime = goRuntime;
    }
    if (appName != null) {
      $result.appName = appName;
    }
    return $result;
  }
  PluginVersionResponse._() : super();
  factory PluginVersionResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory PluginVersionResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'PluginVersionResponse', createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'appVersion')
    ..aOS(2, _omitFieldNames ? '' : 'goRuntime')
    ..aOS(3, _omitFieldNames ? '' : 'appName')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  PluginVersionResponse clone() => PluginVersionResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  PluginVersionResponse copyWith(void Function(PluginVersionResponse) updates) => super.copyWith((message) => updates(message as PluginVersionResponse)) as PluginVersionResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginVersionResponse create() => PluginVersionResponse._();
  PluginVersionResponse createEmptyInstance() => create();
  static $pb.PbList<PluginVersionResponse> createRepeated() => $pb.PbList<PluginVersionResponse>();
  @$core.pragma('dart2js:noInline')
  static PluginVersionResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<PluginVersionResponse>(create);
  static PluginVersionResponse? _defaultInstance;

  /// app_version is the version of the application.
  @$pb.TagNumber(1)
  $core.String get appVersion => $_getSZ(0);
  @$pb.TagNumber(1)
  set appVersion($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasAppVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearAppVersion() => clearField(1);

  /// go_runtime is the Go version the server was compiled with.
  @$pb.TagNumber(2)
  $core.String get goRuntime => $_getSZ(1);
  @$pb.TagNumber(2)
  set goRuntime($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasGoRuntime() => $_has(1);
  @$pb.TagNumber(2)
  void clearGoRuntime() => clearField(2);

  /// app_name is the name of the underlying app running the server.
  @$pb.TagNumber(3)
  $core.String get appName => $_getSZ(2);
  @$pb.TagNumber(3)
  set appName($core.String v) { $_setString(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasAppName() => $_has(2);
  @$pb.TagNumber(3)
  void clearAppName() => clearField(3);
}

class PluginCallActionStreamRequest extends $pb.GeneratedMessage {
  factory PluginCallActionStreamRequest({
    $core.String? user,
    $core.String? action,
    $core.List<$core.int>? data,
  }) {
    final $result = create();
    if (user != null) {
      $result.user = user;
    }
    if (action != null) {
      $result.action = action;
    }
    if (data != null) {
      $result.data = data;
    }
    return $result;
  }
  PluginCallActionStreamRequest._() : super();
  factory PluginCallActionStreamRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory PluginCallActionStreamRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'PluginCallActionStreamRequest', createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'user')
    ..aOS(2, _omitFieldNames ? '' : 'action')
    ..a<$core.List<$core.int>>(3, _omitFieldNames ? '' : 'data', $pb.PbFieldType.OY)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  PluginCallActionStreamRequest clone() => PluginCallActionStreamRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  PluginCallActionStreamRequest copyWith(void Function(PluginCallActionStreamRequest) updates) => super.copyWith((message) => updates(message as PluginCallActionStreamRequest)) as PluginCallActionStreamRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginCallActionStreamRequest create() => PluginCallActionStreamRequest._();
  PluginCallActionStreamRequest createEmptyInstance() => create();
  static $pb.PbList<PluginCallActionStreamRequest> createRepeated() => $pb.PbList<PluginCallActionStreamRequest>();
  @$core.pragma('dart2js:noInline')
  static PluginCallActionStreamRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<PluginCallActionStreamRequest>(create);
  static PluginCallActionStreamRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get user => $_getSZ(0);
  @$pb.TagNumber(1)
  set user($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasUser() => $_has(0);
  @$pb.TagNumber(1)
  void clearUser() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get action => $_getSZ(1);
  @$pb.TagNumber(2)
  set action($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasAction() => $_has(1);
  @$pb.TagNumber(2)
  void clearAction() => clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get data => $_getN(2);
  @$pb.TagNumber(3)
  set data($core.List<$core.int> v) { $_setBytes(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasData() => $_has(2);
  @$pb.TagNumber(3)
  void clearData() => clearField(3);
}

class PluginCallActionStreamResponse extends $pb.GeneratedMessage {
  factory PluginCallActionStreamResponse({
    $core.List<$core.int>? response,
  }) {
    final $result = create();
    if (response != null) {
      $result.response = response;
    }
    return $result;
  }
  PluginCallActionStreamResponse._() : super();
  factory PluginCallActionStreamResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory PluginCallActionStreamResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'PluginCallActionStreamResponse', createEmptyInstance: create)
    ..a<$core.List<$core.int>>(1, _omitFieldNames ? '' : 'response', $pb.PbFieldType.OY)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  PluginCallActionStreamResponse clone() => PluginCallActionStreamResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  PluginCallActionStreamResponse copyWith(void Function(PluginCallActionStreamResponse) updates) => super.copyWith((message) => updates(message as PluginCallActionStreamResponse)) as PluginCallActionStreamResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginCallActionStreamResponse create() => PluginCallActionStreamResponse._();
  PluginCallActionStreamResponse createEmptyInstance() => create();
  static $pb.PbList<PluginCallActionStreamResponse> createRepeated() => $pb.PbList<PluginCallActionStreamResponse>();
  @$core.pragma('dart2js:noInline')
  static PluginCallActionStreamResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<PluginCallActionStreamResponse>(create);
  static PluginCallActionStreamResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<$core.int> get response => $_getN(0);
  @$pb.TagNumber(1)
  set response($core.List<$core.int> v) { $_setBytes(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasResponse() => $_has(0);
  @$pb.TagNumber(1)
  void clearResponse() => clearField(1);
}

class PluginCallActionUpdate extends $pb.GeneratedMessage {
  factory PluginCallActionUpdate({
    $core.String? updateMessage,
  }) {
    final $result = create();
    if (updateMessage != null) {
      $result.updateMessage = updateMessage;
    }
    return $result;
  }
  PluginCallActionUpdate._() : super();
  factory PluginCallActionUpdate.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory PluginCallActionUpdate.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'PluginCallActionUpdate', createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'updateMessage')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  PluginCallActionUpdate clone() => PluginCallActionUpdate()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  PluginCallActionUpdate copyWith(void Function(PluginCallActionUpdate) updates) => super.copyWith((message) => updates(message as PluginCallActionUpdate)) as PluginCallActionUpdate;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginCallActionUpdate create() => PluginCallActionUpdate._();
  PluginCallActionUpdate createEmptyInstance() => create();
  static $pb.PbList<PluginCallActionUpdate> createRepeated() => $pb.PbList<PluginCallActionUpdate>();
  @$core.pragma('dart2js:noInline')
  static PluginCallActionUpdate getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<PluginCallActionUpdate>(create);
  static PluginCallActionUpdate? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get updateMessage => $_getSZ(0);
  @$pb.TagNumber(1)
  set updateMessage($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasUpdateMessage() => $_has(0);
  @$pb.TagNumber(1)
  void clearUpdateMessage() => clearField(1);
}

/// Define the request and response messages for SendInput
class PluginInputRequest extends $pb.GeneratedMessage {
  factory PluginInputRequest({
    $core.String? user,
    $core.List<$core.int>? data,
  }) {
    final $result = create();
    if (user != null) {
      $result.user = user;
    }
    if (data != null) {
      $result.data = data;
    }
    return $result;
  }
  PluginInputRequest._() : super();
  factory PluginInputRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory PluginInputRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'PluginInputRequest', createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'user')
    ..a<$core.List<$core.int>>(3, _omitFieldNames ? '' : 'data', $pb.PbFieldType.OY)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  PluginInputRequest clone() => PluginInputRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  PluginInputRequest copyWith(void Function(PluginInputRequest) updates) => super.copyWith((message) => updates(message as PluginInputRequest)) as PluginInputRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginInputRequest create() => PluginInputRequest._();
  PluginInputRequest createEmptyInstance() => create();
  static $pb.PbList<PluginInputRequest> createRepeated() => $pb.PbList<PluginInputRequest>();
  @$core.pragma('dart2js:noInline')
  static PluginInputRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<PluginInputRequest>(create);
  static PluginInputRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get user => $_getSZ(0);
  @$pb.TagNumber(1)
  set user($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasUser() => $_has(0);
  @$pb.TagNumber(1)
  void clearUser() => clearField(1);

  @$pb.TagNumber(3)
  $core.List<$core.int> get data => $_getN(1);
  @$pb.TagNumber(3)
  set data($core.List<$core.int> v) { $_setBytes(1, v); }
  @$pb.TagNumber(3)
  $core.bool hasData() => $_has(1);
  @$pb.TagNumber(3)
  void clearData() => clearField(3);
}

class PluginInputResponse extends $pb.GeneratedMessage {
  factory PluginInputResponse({
    $core.bool? success,
    $core.String? message,
  }) {
    final $result = create();
    if (success != null) {
      $result.success = success;
    }
    if (message != null) {
      $result.message = message;
    }
    return $result;
  }
  PluginInputResponse._() : super();
  factory PluginInputResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory PluginInputResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'PluginInputResponse', createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'success')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  PluginInputResponse clone() => PluginInputResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  PluginInputResponse copyWith(void Function(PluginInputResponse) updates) => super.copyWith((message) => updates(message as PluginInputResponse)) as PluginInputResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginInputResponse create() => PluginInputResponse._();
  PluginInputResponse createEmptyInstance() => create();
  static $pb.PbList<PluginInputResponse> createRepeated() => $pb.PbList<PluginInputResponse>();
  @$core.pragma('dart2js:noInline')
  static PluginInputResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<PluginInputResponse>(create);
  static PluginInputResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get success => $_getBF(0);
  @$pb.TagNumber(1)
  set success($core.bool v) { $_setBool(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasSuccess() => $_has(0);
  @$pb.TagNumber(1)
  void clearSuccess() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => clearField(2);
}


const _omitFieldNames = $core.bool.fromEnvironment('protobuf.omit_field_names');
const _omitMessageNames = $core.bool.fromEnvironment('protobuf.omit_message_names');
