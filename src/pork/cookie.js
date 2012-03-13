goog.provide('pork.cookies');

/**
@return {Object.<string, string>}
*/
pork.cookies.get = function() {
  var p = document.cookie.split(';');
  var r = {};
  for (var i = 0, n = p.length; i < n; ++i) {
    if (p[i].length == 0)
      continue;
    var q = p[i].split('=');
    if (q.length != 2)
      continue;
    r[q[0].trim()] = q[1];
  }
  return r;
};

/**
@param {string} name
@param {string} value
@param {number} secondsUntilExpiry
*/
pork.cookies.set = function(name, value, secondsUntilExpiry) {
};