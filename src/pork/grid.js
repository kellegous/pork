(function(){

var layout = function(element) {
  return element.offsetLeft;  
};

var css = function(element, attrs) {
  for (var i in attrs)
    element.style.setProperty(i, attrs[i], '');
  return element;
}

/**
  @constructor
  @param offsets {Array.<int>}
  @param doc {Document}
*/
var Grid = function(x, y, doc) {
  var self = this;
  var view = doc.documentElement;
  var grid = css(doc.createElement('div'), {
    'position': 'absolute',
    'top': '0',
    'left': '0',
    'width': view.offsetWidth + 'px',
    'height': view.offsetHeight + 'px',
    'pointer-events': 'none',
    'z-index': '10000',
    '-webkit-transition': 'opacity 500ms ease-in-out',
    'opacity': '0'
  });

  var left = 0;
  for (var i = 0, n = x.length; i < n; ++i) {
    left += x[i];
    var line = css(grid.appendChild(doc.createElement('div')), {
      'position': 'absolute',
      'border-left': '1px dotted rgba(0, 0, 0, 0.8)',
      'top': '0',
      'bottom': '0',
      'width': '10px'
    });
    line.style.left = left + 'px';
  }

  var top = 0;
  for (var i = 0, n = y.length; i < n; ++i) {
    top += y[i];
    var line = css(grid.appendChild(doc.createElement('div')), {
      'position': 'absolute',
      'border-top': '1px dotted rgba(0, 0, 0, 0.5)',
      'left': '0',
      'right': '0',
      'height': '10px'
    });
    line.style.top = top + 'px';
  }

  var visible = false;
  var body = doc.body;

  // listen for opacity transition to complete.
  grid.addEventListener('webkitTransitionEnd', function(e) {
    if (grid.style.opacity === '0')
      body.removeChild(grid);
  }, false);

  // listen for Cmd-Opt to get released.
  body.addEventListener('keyup', function(e) {
    if (!visible)
      return;
    if (e.keyIdentifier === 'Alt' && !e.metaKey
        || e.keyIdentifier === 'Meta' && !e.altKey) {
      visible = false;
      grid.style.opacity = '0';
    }
  }, false);

  // listen for Cmd-Opt to get depressed.
  body.addEventListener('keydown', function(e) {
    if (visible)
      return;
    if (e.keyIdentifier === 'Alt' && e.metaKey
        || e.keyIdentifier === 'Meta' && e.altKey) {
      visible = true;
      layout(body.appendChild(grid));
      css(grid, {
        'opacity': '1',
        'z-index': '10000',
        'width': Math.max(view.offsetWidth, window.innerWidth) + 'px',
        'height': Math.max(view.offsetHeight, window.innerHeight) + 'px'
      });
      grid.style.opacity = '1';
    }
  }, false);
};

/**
@private
@type {bool}
*/
Grid.prototype._visible = false;

/**
 @param offsets {Array.<int>}
 @param doc {?Document}
*/
pork.grid = function(x, y, doc) {
  return new Grid(x, y, doc || document);
};

})();