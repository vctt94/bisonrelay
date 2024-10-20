import 'dart:async';
import 'dart:collection';
import 'package:bruig/models/menus.dart';
import 'package:bruig/models/resources.dart';
import 'package:flutter/foundation.dart';
import 'package:golib_plugin/definitions.dart';
import 'package:golib_plugin/golib_plugin.dart';
import '../storage_manager.dart';

const SCE_unknown = 0;
const SCE_sending = 1;
const SCE_sent = 2;
const SCE_received = 3;
const SCE_errored = 99;

class SynthChatEvent extends ChatEvent with ChangeNotifier {
  SynthChatEvent(String msg, [this._state = SCE_unknown, this._error])
      : super("", msg);

  int _state;
  int get state => _state;
  void set state(int v) {
    _state = v;
    notifyListeners();
  }

  Exception? _error;
  Exception? get error => _error;
  void set error(Exception? e) {
    if (e == null) throw Exception("Cannot set error to null");
    _error = e;
    _state = SCE_errored;
    notifyListeners();
  }
}

class RequestedResourceEvent extends ChatEvent {
  final PagesSession session;

  RequestedResourceEvent(String uid, this.session)
      : super(uid, "Fetching user resources");
}

const int CMS_unknown = 0;
const int CMS_sending = 1;
const int CMS_sent = 2;
const int CMS_errored = 3;
const int CMS_canceled = 4;

const int Suggestion_received = 0;
const int Suggestion_accepted = 1;
const int Suggestion_confirmed = 2;
const int Suggestion_canceled = 3;
const int Suggestion_errored = 4;
const int Suggestion_alreadyKnown = 5;

class ChatEventModel extends ChangeNotifier {
  final ChatEvent event;
  final ChatModel? source; // null if it's from the local client.
  ChatEventModel(this.event, this.source);

  int _sentState = CMS_unknown;
  int get sentState => _sentState;
  void set sentState(int v) {
    _sentState = v;
    notifyListeners();
  }

  String? _sendError;
  String? get sendError => _sendError;
  void set sendError(String? err) {
    _sendError = err;
    _sentState = CMS_errored;
    notifyListeners();
  }

  bool _firstUnread = false;
  bool get firstUnread => _firstUnread;
  void set firstUnread(bool b) {
    _firstUnread = b;
    notifyListeners();
  }

  bool _sameUser = false;
  bool get sameUser => _sameUser;
  void set sameUser(bool b) {
    _sameUser = b;
    notifyListeners();
  }
}

class ChatModel extends ChangeNotifier {
  final String id; // RemoteUID or GC ID
  final bool isGC;

  String _nick; // Nick or GC name
  String get nick => _nick;
  void set nick(String nn) {
    _nick = nn;
    notifyListeners();
  }

  ChatModel(this.id, this._nick, this.isGC);

  int _unreadMsgCount = 0;
  int get unreadMsgCount => _unreadMsgCount;
  int _unreadEventCount = 0;
  int get unreadEventCount => _unreadEventCount;

  bool _active = false;
  bool get active => _active;
  void _setActive(bool b) {
    _active = b;
    _unreadMsgCount = 0;
    _unreadEventCount = 0;
    notifyListeners();
  }

  bool _showChatListing = true; // Nick or GC name
  bool get showChatListing => _showChatListing;
  set showChatListing(bool b) {
    _showChatListing = b;
    notifyListeners();
  }

  // return the first unread msg index and -1 if there aren't
  // unread msgs
  int firstUnreadIndex() {
    for (int i = 0; i < _msgs.length; i++) {
      if (_msgs[i].firstUnread) {
        return i;
      }
    }
    return -1;
  }

  List<ChatEventModel> _msgs = [];
  UnmodifiableListView<ChatEventModel> get msgs => UnmodifiableListView(_msgs);
  void append(ChatEventModel msg) {
    if (!_active && _unreadMsgCount == 0 && _msgs.isNotEmpty) {
      msg.firstUnread = true;
    }
    if (_msgs.isNotEmpty &&
        _msgs[_msgs.length - 1].source?.id == msg.source?.id) {
      msg.sameUser = true;
    }
    _msgs.add(msg);
    if (!_active) {
      if (msg.event is PM || msg.event is GCMsg) {
        _unreadMsgCount += 1;
      } else {
        _unreadEventCount += 1;
      }
    }
    notifyListeners();
  }

  void appendHistory(ChatEventModel msg) {
    if (_msgs.isNotEmpty &&
        _msgs[_msgs.length - 1].source?.id == msg.source?.id) {
      msg.sameUser = true;
    }
    _msgs.add(msg);
    notifyListeners();
  }

  void removeFirstUnread() {
    for (int i = 0; i < _msgs.length; i++) {
      if (_msgs[i].firstUnread) {
        _msgs[i].firstUnread = false;
        return;
      }
    }
  }

  void payTip(double amount) async {
    var tip = await Golib.payTip(id, amount);
    _msgs.add(ChatEventModel(tip, this));
    notifyListeners();
  }

  Future<void> sendMsg(String msg) async {
    // This may be triggered by autmation sending messages when the chat window
    // is not focused (for example, simplestore placed orders).
    if (!active) {
      _unreadMsgCount += 1;
    }

    if (isGC) {
      var m = GCMsg(id, nick, msg, DateTime.now().millisecondsSinceEpoch);
      var evnt = ChatEventModel(m, null);
      evnt.sentState = CMS_sending; // Track individual sending status?
      if (_msgs.isNotEmpty && _msgs[_msgs.length - 1].source == null) {
        evnt.sameUser = true;
      }
      _msgs.add(evnt);
      notifyListeners();

      try {
        await Golib.sendToGC(id, msg);
        evnt.sentState = CMS_sent;
      } catch (exception) {
        evnt.sendError = "$exception";
      }
    } else {
      var ts = DateTime.now().millisecondsSinceEpoch;
      var m = PM(id, msg, true, ts);
      var evnt = ChatEventModel(m, null);
      evnt.sentState = CMS_sending;
      if (_msgs.isNotEmpty && _msgs[_msgs.length - 1].source == null) {
        evnt.sameUser = true;
      }
      _msgs.add(evnt);
      notifyListeners();

      try {
        await Golib.pm(m);
        evnt.sentState = CMS_sent;
      } catch (exception) {
        evnt.sendError = "$exception";
      }
    }

    // This may be triggered by autmation sending messages when the chat window
    // is not focused (for example, simplestore placed orders).
    if (!active) {
      _unreadMsgCount += 1;
      notifyListeners();
    }
  }

  String workingMsg = "";

  void subscribeToPosts() {
    var event = SynthChatEvent("Subscribing to user's posts");
    event.state = SCE_sending;
    append(ChatEventModel(event, null));
    (() async {
      try {
        await Golib.subscribeToPosts(id);
        event.state = SCE_sent;
      } catch (error) {
        event.error = Exception(error);
      }
    })();
  }

  Future<void> unsubscribeToPosts() {
    var event = SynthChatEvent("Unsubscribing from user's posts");
    event.state = SCE_sending;
    append(ChatEventModel(event, null));
    return (() async {
      try {
        await Golib.unsubscribeToPosts(id);
        event.state = SCE_sent;
      } catch (error) {
        event.error = Exception(error);
      }
    })();
  }

  void requestKXReset() {
    var event = SynthChatEvent("Requesting KX reset", SCE_sending);
    append(ChatEventModel(event, null));
    (() async {
      try {
        await Golib.requestKXReset(id);
        event.state = SCE_sent;
      } catch (error) {
        event.error = new Exception(error);
      }
    })();
  }

  Future<void> resendGCList() async => await Golib.resendGCList(id);
}

class ClientModel extends ChangeNotifier {
  ClientModel() {
    _handleAcceptedInvites();
    _handleChatMsgs();
    readAddressBook();
    _handleServerSessChanged();
    _handleGCListUpdates();
    _fetchInfo();
    _handleSSOrders();
  }

  List<ChatModel> _gcChats = [];
  UnmodifiableListView<ChatModel> get gcChats => UnmodifiableListView(_gcChats);

  void set gcChats(List<ChatModel> gc) {
    _gcChats = gc;
    notifyListeners();
  }

  List<ChatModel> _userChats = [];
  UnmodifiableListView<ChatModel> get userChats =>
      UnmodifiableListView(_userChats);

  void set userChats(List<ChatModel> us) {
    _userChats = us;
    notifyListeners();
  }

  List<ChatModel> _filteredSearch = [];
  UnmodifiableListView<ChatModel> get filteredSearch =>
      UnmodifiableListView(_filteredSearch);

  set filteredSearch(List<ChatModel> us) {
    _filteredSearch = us;
    notifyListeners();
  }

  String _filteredSearchString = "";
  String get filteredSearchString => _filteredSearchString;
  set filteredSearchString(String b) {
    _filteredSearch = [];
    _filteredSearchString = b;
    if (b != "") {
      for (int i = 0; i < _gcChats.length; i++) {
        if (_gcChats[i].nick.toLowerCase().contains(b.toLowerCase())) {
          _filteredSearch.add(_gcChats[i]);
        }
      }
      for (int i = 0; i < _hiddenGCs.length; i++) {
        if (_hiddenGCs[i].nick.toLowerCase().contains(b.toLowerCase())) {
          _filteredSearch.add(_hiddenGCs[i]);
        }
      }
      for (int i = 0; i < _userChats.length; i++) {
        if (_userChats[i].nick.toLowerCase().contains(b.toLowerCase())) {
          _filteredSearch.add(_userChats[i]);
        }
      }
      for (int i = 0; i < _hiddenUsers.length; i++) {
        if (_hiddenUsers[i].nick.toLowerCase().contains(b.toLowerCase())) {
          _filteredSearch.add(_hiddenUsers[i]);
        }
      }
    }
    _filteredSearch
        .sort((a, b) => a._nick.toLowerCase().compareTo(b._nick.toLowerCase()));
    notifyListeners();
  }

  List<ChatModel> _hiddenGCs = [];
  UnmodifiableListView<ChatModel> get hiddenGCs =>
      UnmodifiableListView(_hiddenGCs);

  set hiddenGCs(List<ChatModel> us) {
    _hiddenGCs = us;
    notifyListeners();
  }

  List<ChatModel> _hiddenUsers = [];
  UnmodifiableListView<ChatModel> get hiddenUsers =>
      UnmodifiableListView(_hiddenUsers);

  set hiddenUsers(List<ChatModel> us) {
    _hiddenUsers = us;
    notifyListeners();
  }

  String _savedHiddenUsers = "";
  String get savedHiddenUsers => _savedHiddenUsers;
  set savedHiddenUsers(String b) {
    _savedHiddenUsers = b;
    notifyListeners();
  }

  String _savedHiddenGCs = "";
  String get savedHiddenGCs => _savedHiddenGCs;
  set savedHiddenGCs(String b) {
    _savedHiddenGCs = b;
    notifyListeners();
  }

  bool _loadingAddressBook = true;
  bool get loadingAddressBook => _loadingAddressBook;
  void set loadingAddressBook(bool b) {
    _loadingAddressBook = b;
    notifyListeners();
  }

  bool _hasUnreadChats = false;
  bool get hasUnreadChats => _hasUnreadChats;
  void set hasUnreadChats(bool b) {
    _hasUnreadChats = b;
    notifyListeners();
  }

  bool _showAddressBook = false;
  bool get showAddressBook => _showAddressBook;
  set showAddressBook(bool b) {
    _showAddressBook = b;
    notifyListeners();
  }

  void showAddressBookScreen() {
    showAddressBook = true;
  }

  void hideAddressBookScreen() {
    showAddressBook = false;
  }

  void startChat(ChatModel chat, bool alreadyOpened) {
    if (!alreadyOpened) {
      if (chat.isGC) {
        _hiddenGCs.remove(chat);
        List<ChatModel> newGcChats = [];
        newGcChats.add(chat);
        for (int i = 0; i < _gcChats.length; i++) {
          newGcChats.add(_gcChats[i]);
        }
        _gcChats = newGcChats;
        _subGCMenus[chat.id] = buildGCMenu(chat);
        if (_savedHiddenGCs.contains(chat.nick)) {
          var savedHiddenGCsSplit = _savedHiddenGCs.split(",");
          var newGCSplitStr = "";
          for (int i = 0; i < savedHiddenGCsSplit.length; i++) {
            if (!savedHiddenGCsSplit[i].contains(chat.nick)) {
              if (newGCSplitStr.isEmpty) {
                newGCSplitStr = chat.nick;
              } else {
                newGCSplitStr += ", ${chat.nick}";
              }
            }
          }
          _savedHiddenGCs = newGCSplitStr;
          StorageManager.saveData('gcHiddenList', _savedHiddenGCs);
        }
      } else {
        _hiddenUsers.remove(chat);
        List<ChatModel> newUserChats = [];
        newUserChats.add(chat);
        for (int i = 0; i < userChats.length; i++) {
          newUserChats.add(_userChats[i]);
        }
        _userChats = newUserChats;
        _subUserMenus[chat.id] = buildUserChatMenu(chat);
        if (_savedHiddenUsers.contains(chat.nick)) {
          var savedHiddenUsersSplit = _savedHiddenUsers.split(",");
          var newUserSplitStr = "";
          for (int i = 0; i < savedHiddenUsersSplit.length; i++) {
            if (!savedHiddenUsersSplit[i].contains(chat.nick)) {
              if (newUserSplitStr.isEmpty) {
                newUserSplitStr = chat.nick;
              } else {
                newUserSplitStr += ", ${chat.nick}";
              }
            }
          }
          _savedHiddenUsers = newUserSplitStr;
          StorageManager.saveData('userHiddenList', _savedHiddenUsers);
        }
      }
    }
    active = chat;
    notifyListeners();
  }

  final Map<String, List<ChatMenuItem>> _subGCMenus = {};
  UnmodifiableMapView<String, List<ChatMenuItem>> get subGCMenus =>
      UnmodifiableMapView(_subGCMenus);

  final Map<String, List<ChatMenuItem>> _subUserMenus = {};
  UnmodifiableMapView<String, List<ChatMenuItem>> get subUserMenus =>
      UnmodifiableMapView(_subUserMenus);

  List<ChatMenuItem> _activeSubMenu = [];
  UnmodifiableListView<ChatMenuItem> get activeSubMenu =>
      UnmodifiableListView(_activeSubMenu);

  void set activeSubMenu(List<ChatMenuItem> sm) {
    _activeSubMenu = sm;
    notifyListeners();
  }

  void showSubMenu(bool isGC, String id) {
    if (isGC) {
      activeSubMenu = subGCMenus[id] ?? [];
    } else {
      activeSubMenu = subUserMenus[id] ?? [];
    }
    notifyListeners();
  }

  void hideSubMenu() {
    activeSubMenu = [];
    notifyListeners();
  }

  String _publicID = "";
  String get publicID => _publicID;

  String _nick = "";
  String get nick => _nick;

  ServerSessionState _connState = ServerSessionState.empty();
  ServerSessionState get connState => _connState;
  bool get isOnline => _connState.state == connStateOnline;

  String _network = "";
  String get network => _network;

  ChatModel? _active;
  ChatModel? get active => _active;

  void set active(ChatModel? c) {
    _profile = null;
    // Remove new posts messages
    _active?.removeFirstUnread();
    _active?._setActive(false);
    _active = c;
    showAddressBook = false;
    c?._setActive(true);

    // Check for unreadMessages so we can turn off sidebar notification
    bool unreadChats = false;
    for (int i = 0; i < _gcChats.length; i++) {
      if (_gcChats[i].unreadMsgCount > 0) {
        unreadChats = true;
      }
    }

    for (int i = 0; i < _userChats.length; i++) {
      if (_userChats[i].unreadMsgCount > 0) {
        unreadChats = true;
      }
    }
    hasUnreadChats = unreadChats;
    hideSubMenu();
    notifyListeners();
  }

  ChatModel? _profile;
  ChatModel? get profile => _profile;
  set profile(ChatModel? c) {
    _profile = c;
    //c?._setShowProfile(true);
    notifyListeners();
  }

  void setActiveByNick(String nick, bool isGC) {
    try {
      var c = isGC
          ? _gcChats.firstWhere((c) => c.nick == nick)
          : _userChats.firstWhere((c) => c.nick == nick);
      active = c;
    } on StateError {
      // Ignore if chat doesn't exist.
    }
  }

  Future<void> newSentMsg(ChatModel? chat) async {
    if (chat != null) {
      if (chat.isGC) {
        _gcChats.remove(chat);
        List<ChatModel> newGcChats = [];
        newGcChats.add(chat);
        for (int i = 0; i < _gcChats.length; i++) {
          newGcChats.add(_gcChats[i]);
        }
        _gcChats = newGcChats;
      } else {
        _userChats.remove(chat);
        List<ChatModel> newUserChats = [];
        newUserChats.add(chat);
        for (int i = 0; i < _userChats.length; i++) {
          newUserChats.add(_userChats[i]);
        }
        _userChats = newUserChats;
      }
    }
    notifyListeners();
  }

  Map<String, ChatModel> _activeChats = Map<String, ChatModel>();
  ChatModel? getExistingChat(String uid) => _activeChats[uid];

  Future<ChatModel> _newChat(
      String id, String alias, bool isGC, bool startup) async {
    alias = alias.trim();

    var c = _activeChats[id];
    if (c != null) {
      if (alias != "" && alias != c.nick) {
        c.nick = alias;
        notifyListeners();
      }
      return c;
    }

    alias = alias == "" ? "[blank]" : alias;
    c = ChatModel(id, alias, isGC);
    _activeChats[id] = c;

    // Start with 500 messages and first page (0). We can load more with a scrolling
    // mechanism in the future
    List<LogEntry> chatHistory = [];
    try {
      chatHistory = await Golib.readChatHistory(id, isGC ? alias : "", 500, 0);
    } catch (exception) {
      // Ignore, as we might be opening a chat for a user that hasn't been fully
      // setup yet.
    }
    for (int i = 0; i < chatHistory.length; i++) {
      ChatEventModel evnt;
      if (isGC) {
        var m = GCMsg(
            id,
            chatHistory[i].from,
            chatHistory[i].message,
            chatHistory[i].timestamp *
                (chatHistory[i].from == _nick ? 1000 : 1));
        evnt = ChatEventModel(
            m,
            chatHistory[i].from != _nick
                ? ChatModel(chatHistory[i].from, chatHistory[i].from, true)
                : null);
      } else {
        var m = PM(
            id,
            chatHistory[i].message,
            chatHistory[i].from == _nick,
            chatHistory[i].timestamp *
                (chatHistory[i].from == _nick ? 1000 : 1));
        evnt = ChatEventModel(
            m,
            chatHistory[i].from != _nick
                ? ChatModel(id, chatHistory[i].from, false)
                : null);
      }
      c.appendHistory(evnt);
    }

    // Sorting algo to attempt to retain order
    int sortUsedChats(ChatModel a, ChatModel b) {
      // First check if either is empty, if so prioritize the non-empty one.
      if (b._msgs.isEmpty) {
        if (a._msgs.isEmpty) {
          return 0;
        } else {
          return -1;
        }
      } else if (a._msgs.isEmpty) {
        return 1;
      }
      // If both are not empty, then check to see if unreadMsgCount is > 0 for
      // either.
      if (b.unreadMsgCount > 0 || a.unreadMsgCount > 0) {
        return b.unreadMsgCount.compareTo(a.unreadMsgCount);
      }
      // If unreadMsgCount are both 0, then check last message timestamps;
      var bTimeStamp = 0;
      var aTimeStamp = 0;
      var bLastMessage = b._msgs[b._msgs.length - 1];
      var bLastMessageEvent = b._msgs[b._msgs.length - 1].event;
      if (bLastMessageEvent is PM) {
        bTimeStamp = bLastMessage.source?.nick == null
            ? bLastMessageEvent.timestamp
            : bLastMessageEvent.timestamp * 1000;
      } else if (bLastMessageEvent is GCMsg) {
        bTimeStamp = bLastMessage.source?.nick == null
            ? bLastMessageEvent.timestamp
            : bLastMessageEvent.timestamp * 1000;
      }

      var aLastMessage = a._msgs[a._msgs.length - 1];
      var aLastMessageEvent = a._msgs[a._msgs.length - 1].event;
      if (aLastMessageEvent is PM) {
        aTimeStamp = aLastMessage.source?.nick == null
            ? aLastMessageEvent.timestamp
            : aLastMessageEvent.timestamp * 1000;
      } else if (aLastMessageEvent is GCMsg) {
        aTimeStamp = aLastMessage.source?.nick == null
            ? aLastMessageEvent.timestamp
            : aLastMessageEvent.timestamp * 1000;
      }
      return bTimeStamp.compareTo(aTimeStamp);
    }

    // TODO: this test should be superflous.
    if (isGC) {
      if (_gcChats.indexWhere((c) => c.id == id) == -1 &&
          ((c._msgs.isNotEmpty && !_savedHiddenGCs.contains(c.nick)) ||
              (c._msgs.isEmpty && !startup))) {
        // Add to list of chats if not empty or the chat is empty and
        // not being create via readAddressBook call.
        _gcChats.add(c);
        _gcChats.sort(sortUsedChats);
        _subGCMenus[c.id] = buildGCMenu(c);
      } else if ((c._msgs.isEmpty || _savedHiddenGCs.contains(c.nick)) &&
          startup) {
        // Add all empty chats on startup to hiddenGCs list.
        _hiddenGCs.add(c);
        _hiddenGCs.sort((a, b) => b.nick.compareTo(a.nick));
      }
    } else {
      if (_userChats.indexWhere((c) => c.id == id) == -1 &&
          ((c._msgs.isNotEmpty && !_savedHiddenUsers.contains(c.nick)) ||
              (c._msgs.isEmpty && !startup))) {
        // Add to list of chats.
        _userChats.add(c);
        _userChats.sort(sortUsedChats);
        _subUserMenus[c.id] = buildUserChatMenu(c);
      } else if ((c._msgs.isEmpty || _savedHiddenUsers.contains(c.nick)) &&
          startup) {
        // Add all empty chats on startup to hiddenGCs list.
        _hiddenUsers.add(c);
        _hiddenUsers.sort((a, b) => b.nick.compareTo(a.nick));
      }
    }

    notifyListeners();

    return c;
  }

  void hideChat(ChatModel chat) {
    if (chat.isGC) {
      _active = null;
      _gcChats.remove(chat);
      _hiddenGCs.add(chat);
      _hiddenUsers.sort((a, b) => b.nick.compareTo(a.nick));
      if (_savedHiddenGCs.isNotEmpty) {
        _savedHiddenGCs += ",${chat.nick}";
      } else {
        _savedHiddenGCs = chat.nick;
      }
      StorageManager.saveData('gcHiddenList', _savedHiddenGCs);
    } else {
      _active = null;
      _userChats.remove(chat);
      _hiddenUsers.add(chat);
      _hiddenUsers.sort((a, b) => b.nick.compareTo(a.nick));
      if (_savedHiddenUsers.isNotEmpty) {
        _savedHiddenUsers += ",${chat.nick}";
      } else {
        _savedHiddenUsers = chat.nick;
      }
      StorageManager.saveData('userHiddenList', _savedHiddenUsers);
    }
    notifyListeners();
  }

  void removeChat(ChatModel chat) {
    if (chat.isGC) {
      _gcChats.remove(chat);
      _subGCMenus.remove(chat.id);
    } else {
      _userChats.remove(chat);
      _subUserMenus.remove(chat.id);
    }
    _activeChats.remove(chat.id);
    notifyListeners();
  }

  String getNick(String uid) {
    var chat = getExistingChat(uid);
    return chat?.nick ?? "";
  }

  void _handleChatMsgs() async {
    var stream = Golib.chatEvents();
    await for (var evnt in stream) {
      if (evnt is FeedPostEvent) {
        if (evnt.sid == publicID) {
          // Ignore own relays.
          continue;
        }
      }
      var isGC = (evnt is GCMsg) || (evnt is GCUserEvent);

      var chat = await _newChat(evnt.sid, "", isGC, false);
      ChatModel? source;
      if (evnt is PM) {
        if (!evnt.mine) {
          source = chat;
        }
      } else if (evnt is GCMsg) {
        source = await _newChat(evnt.senderUID, "", false, false);
      } else if (evnt is GCUserEvent) {
        source = await _newChat(evnt.uid, "", false, false);
      } else {
        source = chat;
      }
      chat.append(ChatEventModel(evnt, source));

      if (chat.isGC) {
        _gcChats.remove(chat);
        List<ChatModel> newGcChats = [];
        newGcChats.add(chat);
        for (int i = 0; i < _gcChats.length; i++) {
          newGcChats.add(_gcChats[i]);
        }
        _gcChats = newGcChats;
      } else {
        _userChats.remove(chat);
        List<ChatModel> newUserChats = [];
        newUserChats.add(chat);
        for (int i = 0; i < _userChats.length; i++) {
          newUserChats.add(_userChats[i]);
        }
        _userChats = newUserChats;
      }
      notifyListeners();
    }
  }

  Future<void> readAddressBook() async {
    await StorageManager.readData('gcHiddenList').then((value) {
      if (value != null && value.length > 0) {
        _savedHiddenGCs = value;
      }
    });
    await StorageManager.readData('userHiddenList').then((value) {
      if (value != null && value.length > 0) {
        _savedHiddenUsers = value;
      } else {}
    });
    var info = await Golib.getLocalInfo();
    _publicID = info.id;
    _nick = info.nick;
    var ab = await Golib.addressBook();
    ab.forEach((v) => _newChat(v.id, v.nick, false, true));
    var gcs = await Golib.listGCs();
    gcs.forEach((v) => _newChat(v.id, v.name, true, true));

    loadingAddressBook = false;
  }

  void acceptInvite(Invitation invite) async {
    var user = await Golib.acceptInvite(invite);
    active = await _newChat(user.uid, user.nick, false, false);
  }

  List<String> _mediating = [];
  bool requestedMediateID(String target) => _mediating.contains(target);
  void requestMediateID(String mediator, String target) async {
    if (!requestedMediateID(target)) {
      _mediating.add(target);
      notifyListeners();
    }
    await Golib.requestMediateID(mediator, target);
  }

  void _fetchInfo() async {
    var res = await Golib.lnGetInfo();
    _network = res.chains[0].network;
  }

  void _handleAcceptedInvites() async {
    var stream = Golib.acceptedInvites();
    await for (var remoteUser in stream) {
      if (requestedMediateID(remoteUser.uid)) {
        _mediating.remove(remoteUser.uid);
      }
      var chat = await _newChat(remoteUser.uid, remoteUser.nick, false, false);
      chat.append(
          ChatEventModel(SynthChatEvent("KX Completed", SCE_received), null));
    }
  }

  void _handleServerSessChanged() async {
    var stream = Golib.serverSessionChanged();
    await for (var state in stream) {
      _connState = state;
      notifyListeners();
    }
  }

  void _handleGCListUpdates() async {
    var stream = Golib.gcListUpdates();
    await for (var update in stream) {
      // Force creating the chat if it doesn't exist.
      _newChat(update.id, update.name, true, false);
    }
  }

  void _handleSSOrderPlaced(SSPlacedOrder po) async {
    try {
      var order = po.order;
      var chat = getExistingChat(order.user);
      if (chat == null) {
        throw "user ${order.user} not found in placed simplestore order";
      }
      chat.sendMsg(po.msg);
    } catch (exception) {
      // TODO: send to snackbar model.
      print("Error while processing SimpleStore order: $exception");
    }
  }

  void _handleSSOrders() async {
    var stream = Golib.simpleStoreOrders();
    await for (var order in stream) {
      _handleSSOrderPlaced(order);
    }
  }
}
