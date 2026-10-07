import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:pulsarules_dart_summary/summary.dart';

const _exitUsage = 64;
const _exitBadData = 65;
const _exitNoInput = 66;
const _exitInternal = 70;

Never _fail(int code, String message) {
  stderr.writeln('summary: $message');
  exit(code);
}

void _discard(File file) {
  try {
    file.deleteSync();
  } on FileSystemException {
    // The caller removes a leftover request file itself.
  }
}

Future<String> _readInput(String path) async {
  final file = File(path);
  final List<int> bytes;
  try {
    bytes = await file.readAsBytes();
  } on FileSystemException catch (error) {
    _fail(_exitNoInput, 'cannot read $path: ${error.message}');
  }
  _discard(file);
  try {
    return utf8.decode(bytes);
  } on FormatException catch (error) {
    _fail(_exitBadData, '$path is not valid UTF-8: ${error.message}');
  }
}

typedef _Request = ({String path, String content, bool test});

List<_Request> _decode(String input) {
  try {
    final requests = jsonDecode(input) as List<Object?>;
    return [
      for (final request in requests)
        (
          path: (request as Map<String, Object?>)['path'] as String,
          content: request['content'] as String,
          test: request['test'] == true,
        ),
    ];
  } on FormatException catch (error) {
    _fail(_exitBadData, 'request is not valid JSON: ${error.message}');
  } on TypeError {
    _fail(
      _exitBadData,
      'request must be a JSON list of {path, content, test} objects',
    );
  }
}

Map<String, Object> _summarizeSafely(_Request request) {
  try {
    return summarize(request.path, request.content, request.test);
  } on Object catch (error) {
    return {'parseError': 'helper failed: ${error.runtimeType}'};
  }
}

Future<void> _run(List<String> arguments) async {
  if (arguments.length != 1) {
    _fail(_exitUsage, 'usage: summary <request.json>');
  }
  final requests = _decode(await _readInput(arguments.single));
  final out = <String, Object>{};
  for (final request in requests) {
    out[request.path] = _summarizeSafely(request);
  }
  stdout.write(jsonEncode(out));
}

void main(List<String> arguments) {
  runZonedGuarded(() => _run(arguments), (error, _) {
    stderr.writeln('summary: internal error: ${error.runtimeType}');
    exit(_exitInternal);
  });
}
