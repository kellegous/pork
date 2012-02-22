var pork = {};

#include "pork/grid.js"

/**
@param url {string}
@param xhrDidSucceed {?function(XMLHttpRequest)}
@param xhrDidFail {?function(XMLHttpRequest)}
*/
var xhrGet = function(url, xhrDidSucceed, xhrDidFail) {
  var xhr = new XMLHttpRequest;
  xhr.open('GET', url, true);
  xhr.onreadystatechange = function() {
    if (xhr.readyState == 4) {
      if (xhr.status == 200) {
        if (xhrDidSucceed)
          xhrDidSucceed(xhr);
      } else {
        if (xhrDidFail)
          xhrDidFail(xhr);
      }
      xhr = null;
    }
  };
  xhr.send(null);
};

/**
 @param c {function()}
*/
pork.whenReady = function(c) {
  if (document.readyState === 'complete') {
    c();
    return;
  }
  document.addEventListener('DOMContentLoaded', c, false);
};