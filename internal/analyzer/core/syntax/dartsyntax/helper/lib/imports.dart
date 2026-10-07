import 'package:analyzer/dart/ast/ast.dart';

import 'lines.dart';

const importGroups = ['dart:', 'package:', 'relative'];

int _group(String uri) {
  if (uri.startsWith('dart:')) return 0;
  return uri.startsWith('package:') ? 1 : 2;
}

List<Map<String, Object>> importsOf(CompilationUnit unit, Lines lines) => [
  for (final directive in unit.directives.whereType<ImportDirective>())
    {
      'literal': directive.uri.toSource(),
      'line': lines.of(directive.firstTokenAfterCommentAndMetadata),
      'group': _group(directive.uri.stringValue ?? ''),
    },
];
