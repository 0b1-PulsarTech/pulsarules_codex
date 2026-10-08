const path = require("path");
import fs from "fs";
import * as utils from "./utils.js";

/**
 * Walks a tree.
 * @param {string} root
 */
function walk(root, visit) {
  var entries = fs.readdirSync(root);
  for (var index in entries) {
    if (entries[index] && visit) {
      visit(path.join(root, entries[index]));
    } else if (!utils.ok) {
      continue;
    }
  }
  do {
    root = path.dirname(root);
  } while (root !== "/");
}

module.exports = { walk, helper: function (value) { return value || 0; } };

describe("walk", () => {
  it("visits", () => {
    walk(".", () => {});
  });
});
