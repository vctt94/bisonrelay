import 'dart:io';

import 'package:shared_preferences/shared_preferences.dart';
import 'package:flutter/foundation.dart';

class StorageManager {
  static const String themeModeKey = "themeMode";
  static const String fontScaleKey = "fontScale";
  static const String goProfilerEnabledKey = "goProfilerEnabled";
  static const String goTimedProfilingKey = "goTimedProfiling";
  static const String ntfnFgSvcKey = "foregroundService";
  static const String notificationsKey = "notifications";
  static const String ntfnPMs = "ntfnPMs";
  static const String ntfnGCMs = "ntfnGCMs";
  static const String ntfnGCMentions = "ntfnGCMentions";
  static const String notifiedGCUnkxdMembers = "notifiedGCUnkdMembers";
  static const String audioCaptureDeviceIdKey = "audioCaptureDeviceId";
  static const String audioPlaybackDeviceIdKey = "audioPlaybackDeviceId";
  static const String showRPCWarningKey = "showRPCWarning";

  static Future<void> saveData(String key, dynamic value) async {
    final prefs = await SharedPreferences.getInstance();
    if (value is int) {
      prefs.setInt(key, value);
    } else if (value is String) {
      prefs.setString(key, value);
    } else if (value is bool) {
      prefs.setBool(key, value);
    } else {
      debugPrint("Invalid Type");
    }
  }

  static Future<dynamic> readData(String key) async {
    final prefs = await SharedPreferences.getInstance();
    dynamic obj = prefs.get(key);
    return obj;
  }

  static Future<bool> exists(String key) async {
    final prefs = await SharedPreferences.getInstance();
    dynamic obj = prefs.containsKey(key);
    return obj;
  }

  static Future<bool> readBool(String key, {defaultVal = false}) async =>
      await readData(key) as bool? ?? defaultVal;
  static Future<void> saveBool(String key, bool value) async =>
      await saveData(key, value);

  static Future<String> readString(String key, {defaultVal = ""}) async =>
      await readData(key) as String? ?? defaultVal;
  static Future<void> saveString(String key, String value) async =>
      await saveData(key, value);

  static Future<bool> deleteData(String key) async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.remove(key);
  }

  static Future<void> setupDefaults() async {
    if (Platform.isAndroid) {
      if ((await StorageManager.readData(StorageManager.ntfnFgSvcKey)
              as bool?) ==
          null) {
        await StorageManager.saveData(StorageManager.ntfnFgSvcKey, true);
      }
    }

    if ((await StorageManager.readData(StorageManager.notificationsKey)
            as bool?) ==
        null) {
      await StorageManager.saveData(StorageManager.notificationsKey, true);
    }
    if ((await StorageManager.readData(StorageManager.showRPCWarningKey)
            as bool?) ==
        null) {
      await StorageManager.saveData(StorageManager.showRPCWarningKey, true);
    }
  }
}
