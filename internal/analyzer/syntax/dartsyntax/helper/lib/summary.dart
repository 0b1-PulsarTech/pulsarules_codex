import 'package:analyzer/dart/analysis/results.dart';
import 'package:analyzer/dart/analysis/utilities.dart';

import 'docs.dart';
import 'elses.dart';
import 'functions.dart';
import 'imports.dart';
import 'lines.dart';
import 'names.dart';
import 'numbers.dart';

Map<String, Object> summarize(String path, String content, bool isTest) {
  final result = parseString(
    content: content,
    path: path,
    throwIfDiagnostics: false,
  );
  final lines = Lines(result.lineInfo);
  final unit = result.unit;
  final functions = FunctionCollector(lines, isTest);
  final names = NameCollector(lines);
  final docs = DocCollector(lines, isTest);
  unit.accept(functions);
  unit.accept(names);
  final elses = ElseCollector(lines);
  final numbers = NumberCollector(lines);
  unit.accept(docs);
  unit.accept(elses);
  unit.accept(numbers);
  return {
    'functions': functions.functions.map((frame) => frame.toJson()).toList(),
    'blocks': functions.blocks,
    'imports': importsOf(unit, lines),
    'importGroups': importGroups,
    'names': names.names,
    'docSites': docs.sites,
    'elses': elses.elses,
    'numbers': numbers.numbers,
    'parseError': _firstError(result, lines),
  };
}

String _firstError(ParseStringResult result, Lines lines) {
  if (result.errors.isEmpty) return '';
  final error = result.errors.first;
  return 'line ${lines.at(error.offset)}: ${error.message}';
}
