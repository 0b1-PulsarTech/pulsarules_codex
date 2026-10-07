import 'dart:math' as math;

class Public {
  void shown() {}
  void _hidden() {}
}

class _Private {
  void buried() {}
}

void top() {}
void _topPrivate() {}

class Base {
  Base({this.key});
  final int? key;
}

class Box extends Base {
  Box(this.value, {super.key, required int plain}) : assert(plain > 0);
  final int value;
}

void labelled() {
  outer:
  for (var index = 0; index < 3; index++) {
    for (;;) {
      break outer;
    }
  }
  math.max(1, 2);
  call(label: 1);
}
