import 'package:bruig/components/containers.dart';
import 'package:bruig/components/text.dart';
import 'package:flutter/material.dart';

class PluginBar extends StatelessWidget {
  final int selectedIndex;
  final Function tabChange;
  const PluginBar(this.tabChange, this.selectedIndex, {Key? key})
      : super(key: key);

  @override
  Widget build(BuildContext context) {
    return SecondarySideMenuList(
      width: 130,
      items: [
        ListTile(
            title: const Txt.S("Plugins"),
            selected: selectedIndex == 0,
            onTap: () => tabChange(0, null)),
        ListTile(
            title: const Txt.S("New Plugin"),
            selected: selectedIndex == 1,
            onTap: () => tabChange(1, null)),
        ListTile(
            title: const Txt.S("Installed Plugins"),
            selected: selectedIndex == 2,
            onTap: () => tabChange(2, null)),
      ],
    );
  }
}
