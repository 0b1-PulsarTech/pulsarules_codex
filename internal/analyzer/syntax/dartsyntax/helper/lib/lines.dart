import 'package:analyzer/dart/ast/ast.dart';
import 'package:analyzer/dart/ast/token.dart';
import 'package:analyzer/source/line_info.dart';

class Lines {
  Lines(this.info);

  final LineInfo info;

  int at(int offset) => info.getLocation(offset).lineNumber;

  int of(Token token) => at(token.offset);

  int endOf(AstNode node) =>
      at(node.end > node.offset ? node.end - 1 : node.end);
}

bool isPrivate(String name) => name.startsWith('_');

bool isBlankName(String name) => name.replaceAll('_', '').isEmpty;
