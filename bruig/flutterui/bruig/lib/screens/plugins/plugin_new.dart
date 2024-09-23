import 'package:flutter/material.dart';
import 'package:bruig/models/client.dart';
import 'package:bruig/models/snackbar.dart';
import 'package:golib_plugin/golib_plugin.dart';
import 'package:provider/provider.dart';
import 'package:file_picker/file_picker.dart';

class NewPluginScreen extends StatefulWidget {
  static const routeName = "/newPlugin";

  final ClientModel client;
  const NewPluginScreen(this.client, {Key? key}) : super(key: key);

  @override
  State<NewPluginScreen> createState() => _NewPluginScreenState();
}

class _NewPluginScreenState extends State<NewPluginScreen> {
  final _formKey = GlobalKey<FormState>();
  String url = '';
  String id = '';
  String certPath = '';
  bool isLoading = false;

  void submitForm() async {
    if (_formKey.currentState?.validate() ?? false) {
      _formKey.currentState?.save();
      setState(() {
        isLoading = true;
      });

      var snackbar = SnackBarModel.of(context);
      try {
        // Call the method to add the plugin here, e.g., Golib.addPlugin(url, certPath);
        await Golib.addNewPlugin(id, url, certPath);
        snackbar.success("Plugin added successfully!");
        Navigator.pop(context);
      } catch (e) {
        snackbar.error("Failed to add plugin: $e");
      } finally {
        setState(() {
          isLoading = false;
        });
      }
    }
  }

  Future<void> pickCertificate() async {
    FilePickerResult? result = await FilePicker.platform.pickFiles();

    if (result != null && result.files.single.path != null) {
      setState(() {
        certPath = result.files.single.path!;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text("New Plugin"),
      ),
      body: Padding(
        padding: const EdgeInsets.all(16.0),
        child: Form(
          key: _formKey,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              TextFormField(
                decoration: const InputDecoration(labelText: "Plugin ID"),
                validator: (value) {
                  if (value == null || value.isEmpty) {
                    return "Please enter a plugin ID";
                  }
                  return null;
                },
                onSaved: (value) {
                  id = value ?? '';
                },
              ),
              TextFormField(
                decoration: const InputDecoration(labelText: "Plugin URL"),
                validator: (value) {
                  if (value == null || value.isEmpty) {
                    return "Please enter a plugin URL";
                  }
                  return null;
                },
                onSaved: (value) {
                  url = value ?? '';
                },
              ),
              const SizedBox(height: 20),
              Row(
                children: [
                  Expanded(
                    child: Text(
                      certPath == null ? "No certificate selected" : certPath!,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                  TextButton(
                    onPressed: pickCertificate,
                    child: const Text("Select Certificate"),
                  ),
                ],
              ),
              const SizedBox(height: 40),
              isLoading
                  ? const Center(child: CircularProgressIndicator())
                  : ElevatedButton(
                      onPressed: submitForm,
                      child: const Text("Add Plugin"),
                    ),
            ],
          ),
        ),
      ),
    );
  }
}
