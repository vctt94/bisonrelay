import 'dart:io';

import 'package:bruig/components/context_menu.dart';
import 'package:bruig/components/snackbars.dart';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter_avif/flutter_avif.dart';
import 'package:path_provider/path_provider.dart';
import 'dart:typed_data';
import 'package:path/path.dart' as path;

import 'package:share_plus/share_plus.dart';

var _suggestedExts = {
  "image/png": "png",
  "image/jpg": "jpg",
  "image/jpeg": "jpg",
  "image/bmp": "bmp",
  "image/gif": "gif",
  "image/webp": "webp",
};

bool _isMobile = Platform.isIOS || Platform.isAndroid;

List<PopupMenuItem> _contextMenuItems = [
  ...(_isMobile
      ? const [PopupMenuItem(value: "share", child: Text("Share Image"))]
      : const [PopupMenuItem(value: "save", child: Text("Save Image"))]),
];

Future<String> _tempImgDir() async {
  bool isMobile = Platform.isIOS || Platform.isAndroid;
  String base = isMobile
      ? (await getApplicationCacheDirectory()).path
      : (await getDownloadsDirectory())?.path ?? "";
  return path.join(base, "feedimages");
}

class ImageDialog extends StatelessWidget {
  final Uint8List imgContent;
  final String type;
  final String? name;
  const ImageDialog(this.imgContent, this.type, {this.name, super.key});

  void contextMenuItemClicked(context, value) async {
    var fname = "";
    if (_suggestedExts.containsKey(type)) {
      fname = name ?? "image.${_suggestedExts[type]}";
    }

    switch (value) {
      case "save":
        fname = await FilePicker.platform.saveFile(
              dialogTitle: "Select filename",
              fileName: fname,
            ) ??
            "";

        if (fname == "") {
          return;
        }

        File(fname).writeAsBytesSync(imgContent);
        showSuccessSnackbar(context, "Written image file $fname");
        break;
      case "share":
        if (fname == "") {
          fname = "image.png";
        }
        var dir = await _tempImgDir();
        if (!Directory(dir).existsSync()) {
          Directory(dir).createSync(recursive: true);
        }
        var fpath = path.join(dir, fname);
        File(fpath).writeAsBytesSync(imgContent);
        Share.shareXFiles([XFile(fpath)], text: fname);
        break;
    }
  }

  @override
  Widget build(BuildContext context) {
    return Dialog(
        child: ContextMenu(
      handleItemTap: (v) => contextMenuItemClicked(context, v),
      items: _contextMenuItems,
      child: Container(
        constraints: const BoxConstraints(maxHeight: 1000, maxWidth: 1000),
        decoration: BoxDecoration(
          image: DecorationImage(
            image: MemoryImage(imgContent),
          ),
        ),
      ),
    ));
  }
}

class AvifDialog extends StatelessWidget {
  final Uint8List imgContent;
  const AvifDialog(this.imgContent, {super.key});

  void contextMenuItemClicked(context, value) async {
    var fname = "image.avif";

    switch (value) {
      case "save":
        fname = await FilePicker.platform.saveFile(
              dialogTitle: "Select filename",
              fileName: fname,
            ) ??
            "";

        if (fname == "") {
          return;
        }

        File(fname).writeAsBytesSync(imgContent);
        showSuccessSnackbar(context, "Written avif file $fname");
        break;
      case "share":
        if (fname == "") {
          fname = "image.avif";
        }
        var dir = await _tempImgDir();
        if (!Directory(dir).existsSync()) {
          Directory(dir).createSync(recursive: true);
        }
        fname = path.join(dir, fname);
        File(fname).writeAsBytesSync(imgContent);
        Share.shareXFiles([XFile(fname)], text: "Image");
        break;
    }
  }

  @override
  Widget build(BuildContext context) {
    return Dialog(
        child: ContextMenu(
      handleItemTap: (v) => contextMenuItemClicked(context, v),
      items: _contextMenuItems,
      child: Container(
        constraints: const BoxConstraints(maxHeight: 1000, maxWidth: 1000),
        decoration: BoxDecoration(
          image: DecorationImage(
            image: AvifImage.memory(imgContent).image,
          ),
        ),
      ),
    ));
  }
}
