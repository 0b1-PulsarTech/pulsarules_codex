import 'package:analyzer/dart/ast/ast.dart';

import 'lines.dart';

List<Map<String, Object>> blockDepths(Block body, Lines lines) {
  final out = <Map<String, Object>>[];
  _Depths(lines, out).statements(body.statements, 0);
  return out;
}

class _Depths {
  _Depths(this.lines, this.out);

  final Lines lines;
  final List<Map<String, Object>> out;

  void statements(List<Statement> list, int depth) {
    for (final statement in list) {
      _statement(statement, depth);
    }
  }

  void _statement(Statement node, int depth) {
    switch (node) {
      case IfStatement():
        _if(node, depth);
      case ForStatement():
        _child(node.body, depth);
      case WhileStatement():
        _child(node.body, depth);
      case DoStatement():
        _child(node.body, depth);
      case SwitchStatement():
        _switch(node, depth);
      case TryStatement():
        _try(node, depth);
      case Block():
        _block(node, depth + 1);
      case LabeledStatement():
        _statement(node.statement, depth);
      default:
        break;
    }
  }

  void _block(Block node, int depth) {
    out.add({'line': lines.of(node.leftBracket), 'depth': depth});
    statements(node.statements, depth);
  }

  void _child(Statement body, int depth) {
    if (body is Block) {
      _block(body, depth + 1);
    } else {
      _statement(body, depth + 1);
    }
  }

  void _if(IfStatement node, int depth) {
    _child(node.thenStatement, depth);
    final other = node.elseStatement;
    if (other is IfStatement) {
      _if(other, depth);
    } else if (other != null) {
      _child(other, depth);
    }
  }

  void _switch(SwitchStatement node, int depth) {
    out.add({'line': lines.of(node.leftBracket), 'depth': depth + 1});
    for (final member in node.members) {
      statements(member.statements, depth + 1);
    }
  }

  void _try(TryStatement node, int depth) {
    _block(node.body, depth + 1);
    for (final clause in node.catchClauses) {
      _block(clause.body, depth + 1);
    }
    final finalizer = node.finallyBlock;
    if (finalizer != null) _block(finalizer, depth + 1);
  }
}
