#ifndef PORK_COOKIES_JS
#define PORK_COOKIES_JS

goog.provide('pork.cookies');

/**
@return {Object.<string, string>}
*/
pork.cookies.get = function(name) {
  var p = document.cookie.split(';');
  for (var i = 0, n = p.length; i < n; ++i) {
    var q = p[i].trimLeft();
    if (q.length == 0)
      continue;
    if (q.indexOf(name) != 0 || q[name.length] != '=')
      continue;
    q = q.split('=');
    ASSERT(q.length == 2);
    return q[1];
  }
  return null;
};

/**
@return {Object.<string, string>}
*/
pork.cookies.getAll = function() {
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

/**
@param {string} name
*/
pork.cookies.remove = function(name) {
}

#endif // PORK_COOKIES_JS