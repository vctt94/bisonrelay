import 'dart:math';
import 'dart:io';

import 'package:bruig/components/copyable.dart';
import 'package:bruig/components/empty_widget.dart';
import 'package:bruig/components/recent_log.dart';
import 'package:bruig/components/snackbars.dart';
import 'package:bruig/components/buttons.dart';
import 'package:bruig/components/collapsable.dart';
import 'package:bruig/components/text.dart';
import 'package:bruig/models/shutdown.dart';
import 'package:bruig/models/snackbar.dart';
import 'package:bruig/screens/about.dart';
import 'package:bruig/config.dart';
import 'package:bruig/main.dart';
import 'package:bruig/models/log.dart';
import 'package:bruig/screens/config_network.dart';
import 'package:bruig/screens/shutdown.dart';
import 'package:bruig/screens/startupscreen.dart';
import 'package:bruig/util.dart';
import 'package:flutter/material.dart';
import 'package:golib_plugin/golib_plugin.dart';
import 'package:path/path.dart' as path;
import 'package:provider/provider.dart';
import 'package:bruig/theme_manager.dart';
import 'package:window_manager/window_manager.dart';

class UnlockLNApp extends StatefulWidget {
  Config cfg;
  final String initialRoute;
  SnackBarModel snackBar;
  UnlockLNApp(this.cfg, this.initialRoute, this.snackBar, {super.key});

  void setCfg(Config c) {
    cfg = c;
  }

  @override
  State<UnlockLNApp> createState() => _UnlockLNAppState();
}

class _UnlockLNAppState extends State<UnlockLNApp> with WindowListener {
  final navkey = GlobalKey<NavigatorState>(debugLabel: "unlock-navigator");
  final isMobile = Platform.isIOS || Platform.isAndroid;

  bool pushedToShutdown = false;
  @override
  void onWindowClose() async {
    var isPreventClose = await windowManager.isPreventClose();
    if (!isPreventClose) return;
    if (!pushedToShutdown) {
      ShutdownScreen.startShutdownFromNavKey(navkey);
      pushedToShutdown = true;
    }
  }

  @override
  void initState() {
    super.initState();
    if (!isMobile) {
      windowManager.addListener(this);
      windowManager.setPreventClose(true);
    }
  }

  @override
  void dispose() {
    if (!isMobile) {
      windowManager.removeListener(this);
    }
    super.dispose();
  }

  Config get cfg => widget.cfg;
  @override
  Widget build(BuildContext context) {
    return Consumer<ThemeNotifier>(
        builder: (context, theme, child) => MaterialApp(
              title: "Connect to Bison Relay",
              navigatorKey: navkey,
              initialRoute: widget.initialRoute,
              theme: theme.theme,
              routes: {
                "/": (context) => _LNUnlockPage(widget.cfg, widget.setCfg),
                ConfigNetworkScreen.routeName: (context) =>
                    const ConfigNetworkScreen(),
                "/sync": (context) => _LNChainSyncPage(widget.cfg),
                '/about': (context) => const AboutScreen(),
                ShutdownScreen.routeName: (context) =>
                    ShutdownScreen(globalLogModel, globalShutdownModel),
              },
              builder: (BuildContext context, Widget? child) => Scaffold(
                body:
                    SnackbarDisplayer(widget.snackBar, child ?? const Empty()),
              ),
            ));
  }
}

class _LNUnlockPage extends StatefulWidget {
  final Config cfg;
  final Function(Config) setCfg;
  const _LNUnlockPage(this.cfg, this.setCfg);

  @override
  State<_LNUnlockPage> createState() => __LNUnlockPageState();
}

class __LNUnlockPageState extends State<_LNUnlockPage> {
  bool loading = false;
  final TextEditingController passCtrl = TextEditingController();
  String _validate = "";
  bool compactingDb = false;
  bool compactingDbErrored = false;
  bool migratingDb = false;

  void logUpdated() {
    var log = globalLogModel;
    if (compactingDb != log.compactingDb ||
        compactingDbErrored != log.compactingDbErrored ||
        migratingDb != log.migratingDb) {
      setState(() {
        compactingDb = log.compactingDb;
        compactingDbErrored = log.compactingDbErrored;
        migratingDb = log.migratingDb;
      });
    }
  }

  @override
  void initState() {
    super.initState();
    globalLogModel.addListener(logUpdated);
  }

  @override
  void dispose() {
    passCtrl.dispose();
    super.dispose();
    globalLogModel.removeListener(logUpdated);
  }

  Future<void> unlock() async {
    var snackbar = SnackBarModel.of(context);
    setState(() {
      loading = true;
      _validate = passCtrl.text.isEmpty ? "Password cannot be empty" : "";
    });
    try {
      // Validation failed so don't even attempt
      if (_validate.isNotEmpty) {
        return;
      }
      var cfg = widget.cfg;
      var rpcHost = await Golib.lnRunDcrlnd(
        cfg.internalWalletDir,
        cfg.network,
        passCtrl.text,
        cfg.proxyaddr,
        cfg.torIsolation,
        cfg.proxyUsername,
        cfg.proxyPassword,
        cfg.circuitLimit,
        cfg.syncFreeList,
        cfg.autoCompact,
        cfg.autoCompactMinAge,
        cfg.lnDebugLevel,
        cfg.lnMaxLogFiles,
      );
      var tlsCert = path.join(cfg.internalWalletDir, "tls.cert");
      var macaroonPath = path.join(cfg.internalWalletDir, "data", "chain",
          "decred", cfg.network, "admin.macaroon");
      widget.setCfg(Config.newWithRPCHost(cfg, rpcHost, tlsCert, macaroonPath));
      pushNavigatorFromState(this, "/sync");
    } catch (exception) {
      if (exception.toString().contains("invalid passphrase")) {
        _validate = "Incorrect password, please try again.";
      } else {
        snackbar.error("Unable to unlock wallet: $exception");
      }
      // Catch error and show error in errorText?
    } finally {
      setState(() {
        loading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    Widget extraInfo = const Empty();
    if (loading && migratingDb) {
      extraInfo = const Text(
        "Upgrading DB. This might take a while.",
        style: TextStyle(color: Colors.amber, fontWeight: FontWeight.w500),
      );
    } else if (loading && compactingDbErrored) {
      extraInfo = const Text(
        "Compacting DB errored. Look at the logs to see the cause.",
        style: TextStyle(color: Colors.red),
      );
    } else if (loading && compactingDb) {
      extraInfo = const Text(
        "Compacting DB. This might take a while.",
        style: TextStyle(color: Colors.amber, fontWeight: FontWeight.w500),
      );
    }

    return Consumer<ThemeNotifier>(
        builder: (context, theme, _) =>
            StartupScreen(childrenWidth: 500, <Widget>[
              const Txt.H("Connect to Bison Relay",
                  textAlign: TextAlign.center),
              const SizedBox(height: 30),
              TextField(
                autofocus: true,
                decoration: InputDecoration(
                    labelText: "Password",
                    errorText: _validate,
                    filled: true,
                    fillColor: theme.colors.surface),
                controller: passCtrl,
                obscureText: true,
                onSubmitted: (value) {
                  if (!loading) {
                    unlock();
                  }
                },
              ),
              const SizedBox(height: 30),
              if (!loading) ...[
                LoadingScreenButton(
                  onPressed: !loading ? unlock : null,
                  text: "Unlock Wallet",
                ),
                const SizedBox(height: 15),
                TextButton(
                    onPressed: () {
                      Navigator.of(context, rootNavigator: true)
                          .pushNamed(ConfigNetworkScreen.routeName);
                    },
                    child: const Text("Network Config"))
              ],
              if (loading) ...[
                const CircularProgressIndicator(value: null, strokeWidth: 2),
                const SizedBox(height: 10),
                extraInfo,
                const SizedBox(height: 10),
                Collapsable("Recent Log",
                    child: ConstrainedBox(
                        constraints:
                            const BoxConstraints(maxHeight: 300, maxWidth: 600),
                        child: Container(
                            margin: const EdgeInsets.all(10),
                            padding: const EdgeInsets.all(10),
                            decoration: const BoxDecoration(
                              borderRadius:
                                  BorderRadius.all(Radius.circular(5)),
                            ),
                            child: LogLines(globalLogModel, maxLines: 15))))
              ],
            ]));
  }
}

class _LNChainSyncPage extends StatefulWidget {
  final Config cfg;
  const _LNChainSyncPage(this.cfg);

  @override
  State<_LNChainSyncPage> createState() => _LNChainSyncPageState();
}

String _formatDuration(Duration d) {
  var parts = [];

  if (d.inHours == 1) {
    parts.add("1 hour");
  } else if (d.inHours > 0) {
    parts.add("${d.inHours} hours");
  }

  if (d.inMinutes % 60 == 1) {
    parts.add("1 minute");
  } else if (d.inMinutes % 60 != 0) {
    parts.add("${d.inMinutes % 60} minutes");
  }

  if (d.inSeconds % 60 == 1) {
    parts.add("1 second");
  } else if (d.inSeconds % 60 != 0) {
    parts.add("${d.inSeconds % 60} seconds");
  }

  return parts.join(", ");
}

class _LNChainSyncPageState extends State<_LNChainSyncPage> {
  int blockHeight = 0;
  String blockHash = "";
  DateTime blockTimestamp = DateTime.fromMicrosecondsSinceEpoch(0);
  double currentTimeStamp = DateTime.now().millisecondsSinceEpoch / 1000;
  bool synced = false;
  static const startBlockTimestamp = 1454907600;
  static const fiveMinBlock = 300;
  double progress = 0;
  final DateTime startTime = DateTime.now();
  int initialHeight = -1;
  Duration elapsed = const Duration();
  String get elapsedStr => _formatDuration(elapsed);
  Duration estimated = const Duration();
  String get estimatedStr => _formatDuration(estimated);

  void readSyncProgress() async {
    var stream = Golib.lnInitChainSyncProgress();
    try {
      await for (var update in stream) {
        setState(() {
          blockHeight = update.blockHeight;
          blockHash = update.blockHash;
          blockTimestamp =
              DateTime.fromMillisecondsSinceEpoch(update.blockTimestamp * 1000);
          synced = update.synced;
          if (initialHeight == -1) {
            initialHeight = update.blockHeight;
          }

          var expectedHeight =
              (currentTimeStamp - startBlockTimestamp) / fiveMinBlock;
          var expectedBlocks = max(expectedHeight - initialHeight, 0);
          var fetchedBlocks = update.blockHeight - initialHeight;
          if (fetchedBlocks > expectedBlocks) {
            progress = 1;
          } else if (expectedBlocks == 0) {
            progress = 0;
          } else {
            progress = fetchedBlocks / expectedBlocks;
          }

          var now = DateTime.now();
          elapsed = now.difference(startTime);
          if (progress > 0) {
            estimated = Duration(
                seconds: (elapsed.inSeconds.toDouble() *
                    (1 - progress) ~/
                    progress));
          }
        });
        if (update.synced) {
          syncCompleted();
        }
      }
    } catch (exception) {
      showErrorSnackbar(this, "Unable to read chain sync updates: $exception");
    }
  }

  @override
  void initState() {
    super.initState();
    readSyncProgress();
  }

  void syncCompleted() async {
    runMainApp(widget.cfg);
  }

  @override
  Widget build(BuildContext context) {
    return Consumer<ThemeNotifier>(
        builder: (context, theme, child) => StartupScreen(childrenWidth: 700, [
              const Txt.H("Setting up Bison Relay"),
              const SizedBox(height: 30),
              const Txt.L("Network Sync"),
              const SizedBox(height: 30),
              Row(children: [
                Expanded(
                    child: ClipRRect(
                        borderRadius:
                            const BorderRadius.all(Radius.circular(5)),
                        child: LinearProgressIndicator(
                            minHeight: 8, value: progress > 1 ? 1 : progress))),
                const SizedBox(width: 20),
                Text(
                    "${((progress > 1 ? 1 : progress) * 100).toStringAsFixed(0)}%")
              ]),
              const SizedBox(height: 10),
              Column(children: [
                Wrap(runSpacing: 5, spacing: 20, children: [
                  if (elapsed.inSeconds > 0)
                    SizedBox(
                        width: 300,
                        child: Txt.S("Elapsed: $elapsedStr",
                            color: TextColor.onSurfaceVariant)),
                  if (estimated.inSeconds > 0)
                    SizedBox(
                        width: 300,
                        child: Txt.S("Estimated complete: in $estimatedStr",
                            color: TextColor.onSurfaceVariant)),
                  SizedBox(
                      width: 160,
                      child: Txt.S("Block Height: $blockHeight",
                          color: TextColor.onSurfaceVariant)),
                  Txt.S("Block Time: ${blockTimestamp.toString()}",
                      color: TextColor.onSurfaceVariant),
                  SizedBox(
                      width: 550,
                      child: Row(children: [
                        const Txt.S("Block hash: ",
                            color: TextColor.onSurfaceVariant),
                        Expanded(
                            child: Copyable.txt(Txt.S(blockHash,
                                color: TextColor.onSurfaceVariant,
                                style: theme.extraTextStyles.monospaced,
                                overflow: TextOverflow.ellipsis))),
                      ])),
                ]),
              ]),
              const SizedBox(height: 10),
              Collapsable("Recent Log",
                  child: Container(
                      height: 300,
                      margin: const EdgeInsets.all(10),
                      padding: const EdgeInsets.all(10),
                      decoration: const BoxDecoration(
                        borderRadius: BorderRadius.all(Radius.circular(5)),
                      ),
                      child: LogLines(globalLogModel, maxLines: 15)))
            ]));
  }
}

Future<void> runUnlockDcrlnd(Config cfg) async {
  final theme = await ThemeNotifier.newNotifierWhenLoaded();
  runApp(MultiProvider(
    providers: [
      ChangeNotifierProvider(create: (c) => SnackBarModel()),
      ChangeNotifierProvider.value(value: theme),
    ],
    child: Consumer<SnackBarModel>(
        builder: (context, snackBar, child) => UnlockLNApp(cfg, "/", snackBar)),
  ));
}

Future<void> runChainSyncDcrlnd(Config cfg) async {
  final theme = await ThemeNotifier.newNotifierWhenLoaded();
  runApp(MultiProvider(
    providers: [
      ChangeNotifierProvider(create: (c) => SnackBarModel()),
      ChangeNotifierProvider.value(value: theme),
    ],
    child: Consumer<SnackBarModel>(
        builder: (context, snackBar, child) =>
            UnlockLNApp(cfg, "/sync", snackBar)),
  ));
}

Future<void> runMovePastWindowsSetup(Config cfg) async {
  final theme = await ThemeNotifier.newNotifierWhenLoaded();
  runApp(MultiProvider(
    providers: [
      ChangeNotifierProvider(create: (c) => SnackBarModel()),
      ChangeNotifierProvider.value(value: theme),
    ],
    child: Consumer<SnackBarModel>(
        builder: (context, snackBar, child) =>
            UnlockLNApp(cfg, "/windowsmove", snackBar)),
  ));
}
