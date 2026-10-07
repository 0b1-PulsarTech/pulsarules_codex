String kind(int value) {
  switch (value) {
    case 1:
      return 'one';
    case _:
      return 'any';
  }
}

String arms(int value) => switch (value) {
      1 => 'one',
      int other when other > 5 => 'big',
      _ when value > 2 => 'guarded',
      _ => 'rest',
    };

String typedCase(Object value) {
  switch (value) {
    case int _:
      return 'int';
    case _:
      return 'any';
  }
}

String typedArm(Object value) => switch (value) {
      int _ => 'int',
      _ => 'any',
    };
