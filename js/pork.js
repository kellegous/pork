var pork = {};

#include "pork-grid.js"

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