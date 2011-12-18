var pork = 
// todo: i'm not sure where to take this. i'm inclined to do something
// very d3 like.
function nodesToArray(nodes, array) {
  for (var i = 0, n = nodes.length; i < n; ++i)
    array.push(nodes[i]);
  return array;
}

function Set(nodes) {
  this.nodes = nodes;
}
Set.prototype.nodes = null;
Set.prototype.findOne = function(s) {
  var nodes = this.nodes;
  for (var i = 0, n = nodes.length; i < n; ++i) {
    var t = nodes[i].querySelector(s);
    if (t)
      return new Set([t]);
  }
  return Set.empty;
}
Set.prototype.findAll = function(s) {
  var a = [];
  for (var i = 0, n = nodes.length; i < n; ++i)
    nodesToArray(nodes[i].querySelectorAll(s), a);
  return a;
}
Set.empty = new Set([]);


