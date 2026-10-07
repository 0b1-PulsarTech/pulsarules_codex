import 'b_local.dart';
import 'package:foo/foo.dart';
import 'dart:io';

/// Adds numbers.
int add(int first, [int second = 0]) => first + second;

void configure(String name, {required int retries, int delay = 1}) {}

class Service {
  Service(this.value) {}
  Service.named() : value = 1;
  factory Service.make() => Service(0);
  final int value;
  int get doubled => value * 2;
  void run(List<int> items) {
    final callback = () => items.length;
    items.forEach((item) => print(item));
  }
}
