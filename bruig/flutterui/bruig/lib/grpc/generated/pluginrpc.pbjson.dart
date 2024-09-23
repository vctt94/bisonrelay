//
//  Generated code. Do not modify.
//  source: pluginrpc.proto
//
// @dart = 2.12

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_final_fields
// ignore_for_file: unnecessary_import, unnecessary_this, unused_import

import 'dart:convert' as $convert;
import 'dart:core' as $core;
import 'dart:typed_data' as $typed_data;

@$core.Deprecated('Use renderRequestDescriptor instead')
const RenderRequest$json = {
  '1': 'RenderRequest',
  '2': [
    {'1': 'data', '3': 1, '4': 1, '5': 12, '10': 'data'},
  ],
};

/// Descriptor for `RenderRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List renderRequestDescriptor = $convert.base64Decode(
    'Cg1SZW5kZXJSZXF1ZXN0EhIKBGRhdGEYASABKAxSBGRhdGE=');

@$core.Deprecated('Use renderResponseDescriptor instead')
const RenderResponse$json = {
  '1': 'RenderResponse',
  '2': [
    {'1': 'data', '3': 1, '4': 1, '5': 9, '10': 'data'},
  ],
};

/// Descriptor for `RenderResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List renderResponseDescriptor = $convert.base64Decode(
    'Cg5SZW5kZXJSZXNwb25zZRISCgRkYXRhGAEgASgJUgRkYXRh');

@$core.Deprecated('Use pluginStartStreamRequestDescriptor instead')
const PluginStartStreamRequest$json = {
  '1': 'PluginStartStreamRequest',
  '2': [
    {'1': 'client_id', '3': 1, '4': 1, '5': 9, '10': 'clientId'},
  ],
};

/// Descriptor for `PluginStartStreamRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginStartStreamRequestDescriptor = $convert.base64Decode(
    'ChhQbHVnaW5TdGFydFN0cmVhbVJlcXVlc3QSGwoJY2xpZW50X2lkGAEgASgJUghjbGllbnRJZA'
    '==');

@$core.Deprecated('Use pluginStartStreamResponseDescriptor instead')
const PluginStartStreamResponse$json = {
  '1': 'PluginStartStreamResponse',
  '2': [
    {'1': 'client_id', '3': 1, '4': 1, '5': 9, '10': 'clientId'},
    {'1': 'started', '3': 2, '4': 1, '5': 8, '10': 'started'},
    {'1': 'message', '3': 3, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `PluginStartStreamResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginStartStreamResponseDescriptor = $convert.base64Decode(
    'ChlQbHVnaW5TdGFydFN0cmVhbVJlc3BvbnNlEhsKCWNsaWVudF9pZBgBIAEoCVIIY2xpZW50SW'
    'QSGAoHc3RhcnRlZBgCIAEoCFIHc3RhcnRlZBIYCgdtZXNzYWdlGAMgASgJUgdtZXNzYWdl');

@$core.Deprecated('Use pluginVersionRequestDescriptor instead')
const PluginVersionRequest$json = {
  '1': 'PluginVersionRequest',
};

/// Descriptor for `PluginVersionRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginVersionRequestDescriptor = $convert.base64Decode(
    'ChRQbHVnaW5WZXJzaW9uUmVxdWVzdA==');

@$core.Deprecated('Use pluginVersionResponseDescriptor instead')
const PluginVersionResponse$json = {
  '1': 'PluginVersionResponse',
  '2': [
    {'1': 'app_version', '3': 1, '4': 1, '5': 9, '10': 'appVersion'},
    {'1': 'go_runtime', '3': 2, '4': 1, '5': 9, '10': 'goRuntime'},
    {'1': 'app_name', '3': 3, '4': 1, '5': 9, '10': 'appName'},
  ],
};

/// Descriptor for `PluginVersionResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginVersionResponseDescriptor = $convert.base64Decode(
    'ChVQbHVnaW5WZXJzaW9uUmVzcG9uc2USHwoLYXBwX3ZlcnNpb24YASABKAlSCmFwcFZlcnNpb2'
    '4SHQoKZ29fcnVudGltZRgCIAEoCVIJZ29SdW50aW1lEhkKCGFwcF9uYW1lGAMgASgJUgdhcHBO'
    'YW1l');

@$core.Deprecated('Use pluginCallActionStreamRequestDescriptor instead')
const PluginCallActionStreamRequest$json = {
  '1': 'PluginCallActionStreamRequest',
  '2': [
    {'1': 'user', '3': 1, '4': 1, '5': 9, '10': 'user'},
    {'1': 'action', '3': 2, '4': 1, '5': 9, '10': 'action'},
    {'1': 'data', '3': 3, '4': 1, '5': 12, '10': 'data'},
  ],
};

/// Descriptor for `PluginCallActionStreamRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginCallActionStreamRequestDescriptor = $convert.base64Decode(
    'Ch1QbHVnaW5DYWxsQWN0aW9uU3RyZWFtUmVxdWVzdBISCgR1c2VyGAEgASgJUgR1c2VyEhYKBm'
    'FjdGlvbhgCIAEoCVIGYWN0aW9uEhIKBGRhdGEYAyABKAxSBGRhdGE=');

@$core.Deprecated('Use pluginCallActionStreamResponseDescriptor instead')
const PluginCallActionStreamResponse$json = {
  '1': 'PluginCallActionStreamResponse',
  '2': [
    {'1': 'response', '3': 1, '4': 1, '5': 12, '10': 'response'},
  ],
};

/// Descriptor for `PluginCallActionStreamResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginCallActionStreamResponseDescriptor = $convert.base64Decode(
    'Ch5QbHVnaW5DYWxsQWN0aW9uU3RyZWFtUmVzcG9uc2USGgoIcmVzcG9uc2UYASABKAxSCHJlc3'
    'BvbnNl');

@$core.Deprecated('Use pluginCallActionUpdateDescriptor instead')
const PluginCallActionUpdate$json = {
  '1': 'PluginCallActionUpdate',
  '2': [
    {'1': 'update_message', '3': 1, '4': 1, '5': 9, '10': 'updateMessage'},
  ],
};

/// Descriptor for `PluginCallActionUpdate`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginCallActionUpdateDescriptor = $convert.base64Decode(
    'ChZQbHVnaW5DYWxsQWN0aW9uVXBkYXRlEiUKDnVwZGF0ZV9tZXNzYWdlGAEgASgJUg11cGRhdG'
    'VNZXNzYWdl');

@$core.Deprecated('Use pluginInputRequestDescriptor instead')
const PluginInputRequest$json = {
  '1': 'PluginInputRequest',
  '2': [
    {'1': 'user', '3': 1, '4': 1, '5': 9, '10': 'user'},
    {'1': 'data', '3': 3, '4': 1, '5': 12, '10': 'data'},
  ],
};

/// Descriptor for `PluginInputRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginInputRequestDescriptor = $convert.base64Decode(
    'ChJQbHVnaW5JbnB1dFJlcXVlc3QSEgoEdXNlchgBIAEoCVIEdXNlchISCgRkYXRhGAMgASgMUg'
    'RkYXRh');

@$core.Deprecated('Use pluginInputResponseDescriptor instead')
const PluginInputResponse$json = {
  '1': 'PluginInputResponse',
  '2': [
    {'1': 'success', '3': 1, '4': 1, '5': 8, '10': 'success'},
    {'1': 'message', '3': 2, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `PluginInputResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginInputResponseDescriptor = $convert.base64Decode(
    'ChNQbHVnaW5JbnB1dFJlc3BvbnNlEhgKB3N1Y2Nlc3MYASABKAhSB3N1Y2Nlc3MSGAoHbWVzc2'
    'FnZRgCIAEoCVIHbWVzc2FnZQ==');

