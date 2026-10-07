int flow(int count, bool flag) {
  final z = 1;
  var total = z;
  if (count > 1 && flag) {
    for (var index = 0; index < count; index++) {
      if (index.isEven) {
        total += index;
      }
    }
  }
  switch (count) {
    case 1:
      total++;
    case 2:
      total--;
    default:
      total = 0;
  }
  try {
    total += 1;
  } catch (e) {
    total = 0;
  }
  return flag ? total : -total;
}
