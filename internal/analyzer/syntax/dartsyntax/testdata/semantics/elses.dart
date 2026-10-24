int sign(int value) {
  if (value < 0) {
    return -1;
  } else {
    return 1;
  }
}

void log(int level) {
  if (level == 0) {
    print('quiet');
  } else if (level == 1) {
    print('normal');
  } else if (level == 2) {
    print('loud');
  } else {
    print('loud');
  }
}

void walk(List<int> items) {
  for (final item in items) {
    if (item < 0) continue;
    else print(item);
    if (item == 0) {
      break;
    } else {
      print(item);
    }
    if (item > 9) throw StateError('too big');
    else print(item);
  }
}

List<int> pick(bool all) => [if (all) 1 else 2];

void chains(bool first, bool second, int? state) {
  if (first) print(1);
  else if (second) return;
  else print(2);
  if (first) return;
  else if (second) return;
  else print(3);
  if (state == null) {
    print(4);
  } else if (second) {
    return;
  } else {
    print(5);
  }
}

void labeled(bool first) {
  mark: if (first) {
    break mark;
  } else {
    print(6);
  }
  outer: for (;;) {
    if (first) {
      break outer;
    } else {
      print(7);
    }
  }
}

void bodies(bool first) {
  if (first) {
    print(8);
    return;
  } else {
    print(9);
  }
  if (first) {
    return;
    print(10);
  } else {
    print(11);
  }
  if (first) {} else {
    print(12);
  }
  try {
    print(13);
  } catch (error) {
    if (first) {
      rethrow;
    } else {
      print(14);
    }
  }
}
