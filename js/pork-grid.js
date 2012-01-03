(function(){

/**
  @constructor
  @param offsets {Array.<int>}
  @param doc {Document}
*/
var Grid = function(offsets, doc) {
  var self = this;
  var grid = doc.createElement('div');
  grid.style.cssText = 'position:fixed;'
      + 'top:0; left: 0; right: 0; bottom: 0;'
      + 'pointer-events:none; z-index: 10000';
  var left = 0;
  for (var i = 0, n = offsets.length; i < n; ++i) {
    left += offsets[i];
    var line = grid.appendChild(doc.createElement('div'));
    line.style.cssText = 'position:absolute;'
        + 'border-left:1px dotted rgba(0, 0, 0, 0.5);'
        + 'top:0;bottom:0';
    line.style.left = left + 'px';
  }

  var visible = false;
  var body = doc.body;
  doc.body.addEventListener('keyup', function(e) {
    if (!visible)
      return;
    if (e.keyIdentifier === 'Alt' && !e.metaKey
        || e.keyIdentifier === 'Meta' && !e.altKey) {
      visible = false;
      body.removeChild(grid);
    }
  }, false);

  doc.body.addEventListener('keydown', function(e) {
    if (e.keyIdentifier === 'Alt' && e.metaKey
        || e.keyIdentifier === 'Meta' && e.altKey) {
      visible = true;
      body.appendChild(grid);
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
pork.grid = function(offsets, doc) {
  return new Grid(offsets, doc || document);
};

})();