#include "pork/base.js"
#include "pork/debug.js"

goog.provide("pork");
#include "pork/grid.js"

/**
@type {boolean}
*/
pork.isWebKit;

/**
@type {boolean}
*/
pork.isGecko;

/**
@type {boolean}
*/
pork.isPresto;

/**
@type {boolean}
*/
pork.isMs;

/**
*/
pork.sniff = function() {
  var ua = window.navigator.userAgent;
  if (ua.indexOf('AppleWebKit') != -1) {
    pork.isWebKit = true;
    return;
  }

  if (ua.indexOf('Gecko') != -1) {
    pork.isGecko = true;
    return;
  }

  if (ua.indexOf('Presto') != -1) {
    pork.isPresto = true;
    return;
  }

  // todo: untested
  if (ua.indexOf('IE') != -1) {
    pork.isMs = true;
    return;
  }
};

/**
@param url {string}
@param xhrDidSucceed {?function(XMLHttpRequest)}
@param xhrDidFail {?function(XMLHttpRequest)}
*/
pork.xhrGet = function(url, xhrDidSucceed, xhrDidFail) {
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

/**
@param {Element} element
@param {function()} didEnd
*/
pork.onTransitionEnd = function(element, didEnd) {
  var type = 'transitionend';
  if (pork.isWebKit)
    type = 'webkitTransitionEnd';
  else if (pork.isPresto)
    type = 'oTransitionEnd';

  var hook = function() {
    element.removeEventListener(type, hook, false);
    didEnd();
  };

  element.addEventListener(type, hook, false);
};