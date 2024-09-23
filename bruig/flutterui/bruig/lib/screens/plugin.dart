import 'dart:async';

import 'package:bruig/components/chat/chat_side_menu.dart';
import 'package:bruig/components/plugin_bar.dart';
import 'package:bruig/components/text.dart';
import 'package:bruig/models/client.dart';
import 'package:bruig/models/uistate.dart';
import 'package:bruig/screens/overview.dart';
import 'package:bruig/screens/plugins/plugin_list.dart';
import 'package:bruig/screens/plugins/plugin_content.dart';
import 'package:bruig/screens/plugins/plugin_new.dart';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:bruig/models/plugin.dart';
import 'package:bruig/components/empty_widget.dart';
import 'package:bruig/models/menus.dart';
import 'package:bruig/theme_manager.dart';

class PluginScreenTitle extends StatelessWidget {
  const PluginScreenTitle({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer2<MainMenuModel, ThemeNotifier>(
        builder: (context, menu, theme, child) {
      if (menu.activePageTab <= 0) {
        return const Txt.L("Plugins");
      }
      var idx =
          pluginScreenSub.indexWhere((e) => e.pageTab == menu.activePageTab);

      return Txt.L("Plugins / ${pluginScreenSub[idx].label}");
    });
  }
}

class PluginScreen extends StatefulWidget {
  static const routeName = '/plugins';

  // Goes to the screen that shows the user's plugins.
  // static void showUsersPlugins(BuildContext context, PluginModel plugin) =>
  //     Navigator.of(context).pushReplacementNamed(PluginScreen.routeName,
  //         arguments: PageTabs(0, null, null,
  //             pluginScreenArgs: PluginContentScreenArgs(plugin)));

  final int tabIndex;
  final MainMenuModel mainMenu;
  const PluginScreen(this.mainMenu, {Key? key, this.tabIndex = 0})
      : super(key: key);

  @override
  State<PluginScreen> createState() => _PluginScreenState();
}

class _PluginScreenState extends State<PluginScreen> {
  PluginContentScreenArgs? showPlugin;
  int tabIndex = 0;
  GlobalKey<NavigatorState> navKey = GlobalKey(debugLabel: "plugin nav key");

  @override
  void initState() {
    super.initState();
    tabIndex = widget.tabIndex;
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();

    // Determine if showing a specific plugin.
    if (ModalRoute.of(context)?.settings.arguments != null) {
      final args = ModalRoute.of(context)!.settings.arguments as PageTabs;
      tabIndex = args.tabIndex;
      setState(() {
        if (args.pluginScreenArgs != null) {
          showPlugin = args.pluginScreenArgs;
        }
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    bool isScreenSmall = checkIsScreenSmall(context);

    return Consumer2<MainMenuModel, ClientModel>(
        builder: (context, menu, client, child) {
      bool hasArgs = ModalRoute.of(context)?.settings.arguments != null;

      return ScreenWithChatSideMenu(
        client,
        Row(
          children: [
            // Show plugin sidebar on large screens
            if (!isScreenSmall && !hasArgs) PluginBar(onItemChanged, tabIndex),
            // Expanded plugin content area
            Expanded(child: activeTab()),
          ],
        ),
      );
    });
  }

  Widget activeTab() {
    switch (tabIndex) {
      case 0:
        if (showPlugin == null) {
          return Consumer<ClientModel>(
              builder: (context, client, child) => PluginListScreen(client));
        } else {
          return PluginContentScreen(
              showPlugin as PluginContentScreenArgs, onItemChanged);
        }
      case 1:
        return Consumer<ClientModel>(
            builder: (context, client, child) => NewPluginScreen(client));
      default:
        return Text("Unknown tab index: $tabIndex");
    }
  }

  void onItemChanged(int index, PluginContentScreenArgs? args) {
    setState(() {
      showPlugin = args;
      tabIndex = index;
    });
    Timer(const Duration(milliseconds: 1),
        () => widget.mainMenu.activePageTab = index);
  }
}
