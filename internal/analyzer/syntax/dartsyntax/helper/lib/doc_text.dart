import 'package:analyzer/dart/ast/ast.dart';
import 'package:analyzer/dart/ast/token.dart';

import 'lines.dart';

const _directives = ['ignore:', 'ignore_for_file:', 'pulsarules:'];

String _body(String lexeme) {
  var text = lexeme.trim();
  for (final prefix in ['///', '/**', '//']) {
    if (text.startsWith(prefix)) {
      text = text.substring(prefix.length);
      break;
    }
  }
  if (text.endsWith('*/')) text = text.substring(0, text.length - 2);
  return text.trim();
}

List<String> _docLines(Comment doc) => [
  for (final token in doc.tokens)
    for (final line in _body(token.lexeme).split('\n'))
      line.replaceFirst(RegExp(r'^\*+'), '').trim(),
];

List<String> _adjacentPlain(Comment doc, Lines lines, Token? chain) {
  final first = lines.of(doc.tokens.first);
  final last = lines.of(doc.tokens.last);
  final found = <String>[];
  for (var token = chain; token != null; token = token.next) {
    final lexeme = token.lexeme;
    final line = lines.of(token);
    if (!lexeme.startsWith('//') || lexeme.startsWith('///')) continue;
    if (line == first - 1 || line == last + 1) found.add(_body(lexeme));
  }
  return found;
}

bool _directive(String line) {
  final text = line.replaceFirst(RegExp(r'^/+'), '').trim();
  return _directives.any(text.startsWith);
}

class DocShape {
  DocShape(this.exempt, this.mixed);

  final bool exempt, mixed;
}

DocShape docShape(Comment doc, Lines lines, Token? chain) {
  final own = _docLines(doc);
  final prose = own
      .where((line) => line.isNotEmpty && !_directive(line))
      .toList();
  final all = [...own, ..._adjacentPlain(doc, lines, chain)];
  final hasDirective = all.any(_directive);
  final deprecated =
      prose.isNotEmpty &&
      prose.first.startsWith('Deprecated:') &&
      own.length <= 3 &&
      !own.contains('');
  final marker = prose.isNotEmpty && prose.first.startsWith('simplification:');
  return DocShape(
    prose.isEmpty || deprecated || marker,
    prose.isNotEmpty && hasDirective,
  );
}
