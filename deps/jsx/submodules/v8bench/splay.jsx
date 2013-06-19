// -*- mode: jsx; jsx-indent-level: 4; indent-tabs-mode: nil; -*-
// Copyright 2009 the V8 project authors. All rights reserved.
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//     * Redistributions of source code must retain the above copyright
//       notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above
//       copyright notice, this list of conditions and the following
//       disclaimer in the documentation and/or other materials provided
//       with the distribution.
//     * Neither the name of Google Inc. nor the names of its
//       contributors may be used to endorse or promote products derived
//       from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

// This benchmark is based on a JavaScript log processing module used
// by the V8 profiler to generate execution time profiles for runs of
// JavaScript applications, and it effectively measures how fast the
// JavaScript engine is at allocating nodes and reclaiming the memory
// used for old nodes. Because of the way splay trees work, the engine
// also has to deal with a lot of changes to the large tree object
// graph.


import "./base.jsx";


class Splay {

    // Configuration.
    static const SPLAY_TREE_SIZE = 8000;
    static const SPLAY_TREE_MODIFICATIONS = 80;
    static const SPLAY_TREE_PAYLOAD_DEPTH = 5;

    function constructor() {

        var splayTree = null : SplayTree;

        function GeneratePayloadTree(depth : int, tag : number) : Map.<variant> {
            if (depth == 0) {
                return {
                    array  : [ 0, 1, 2, 3, 4, 5, 6, 7, 8, 9 ] as variant,
                    string : ('String for key ' + tag as string + ' in leaf node') as variant
                };
            } else {
                return {
                    left:  GeneratePayloadTree(depth - 1, tag) as variant,
                    right: GeneratePayloadTree(depth - 1, tag) as variant
                };
            }
        }

        function GenerateKey() : number {
            // The benchmark framework guarantees that random is
            // deterministic; see base.js.
            return BenchmarkUtil.random();
        }

        function InsertNewNode() : number {
            // Insert new node with a unique key.
            var key;
            do {
                key = GenerateKey();
            } while (splayTree.find(key) != null);
            var payload = GeneratePayloadTree(Splay.SPLAY_TREE_PAYLOAD_DEPTH, key);
            splayTree.insert(key, payload);
            return key;
        }

        function SplaySetup() : void {
            splayTree = new SplayTree();
            for (var i = 0; i < Splay.SPLAY_TREE_SIZE; i++) InsertNewNode();
        }

        function SplayTearDown() : void {
            // Allow the garbage collector to reclaim the memory
            // used by the splay tree no matter how we exit the
            // tear down function.
            var keys = splayTree.exportKeys();
            splayTree = null;

            // Verify that the splay tree has the right size.
            var length = keys.length;
            if (length != Splay.SPLAY_TREE_SIZE) {
                throw new Error("Splay tree has wrong size");
            }

            // Verify that the splay tree has sorted, unique keys.
            for (var i = 0; i < length - 1; i++) {
                if (keys[i] >= keys[i + 1]) {
                    throw new Error("Splay tree not sorted");
                }
            }
        }

        function SplayRun() : void {
            // Replace a few nodes in the splay tree.
            for (var i = 0; i < Splay.SPLAY_TREE_MODIFICATIONS; i++) {
                var key = InsertNewNode();
                var greatest = splayTree.findGreatestLessThan(key);
                if (greatest == null) splayTree.remove(key);
                else splayTree.remove(greatest.key);
            }
        }

        var splay = new BenchmarkSuite('Splay', 81491, [
            new Benchmark("Splay", SplayRun, SplaySetup, SplayTearDown)
            ]);
    }
}


class SplayTree {

    /**
     * Pointer to the root node of the tree.
     *
     * @type {SplayTree.Node}
     * @private
     */
    var root_ = null : SplayTreeNode;

    /**
     * Constructs a Splay tree.  A splay tree is a self-balancing binary
     * search tree with the additional property that recently accessed
     * elements are quick to access again. It performs basic operations
     * such as insertion, look-up and removal in O(log(n)) amortized time.
     *
     * @constructor
     */
    function constructor() {
    }



    /**
     * @return {boolean} Whether the tree is empty.
     */
    function isEmpty() : boolean {
        return !this.root_;
    }


    /**
     * Inserts a node into the tree with the specified key and value if
     * the tree does not already contain a node with the specified key. If
     * the value is inserted, it becomes the root of the tree.
     *
     * @param key Key to insert into the tree.
     * @param value Value to insert into the tree.
     */
    function insert(key : number, value : variant) : void {
        if (this.isEmpty()) {
            this.root_ = new SplayTreeNode(key, value);
            return;
        }
        // Splay on the key to move the last node on the search path for
        // the key to the root of the tree.
        this.splay_(key);
        if (this.root_.key == key) {
            return;
        }
        var node = new SplayTreeNode(key, value);
        if (key > this.root_.key) {
            node.left = this.root_;
            node.right = this.root_.right;
            this.root_.right = null;
        } else {
            node.right = this.root_;
            node.left = this.root_.left;
            this.root_.left = null;
        }
        this.root_ = node;
    }


    /**
     * Removes a node with the specified key from the tree if the tree
     * contains a node with this key. The removed node is returned. If the
     * key is not found, an exception is thrown.
     *
     * @param key Key to find and remove from the tree.
     * @return {SplayTree.Node} The removed node.
     */
    function remove(key : number) : SplayTreeNode {
        if (this.isEmpty()) {
            throw new Error('Key not found: ' + key as string);
        }
        this.splay_(key);
        if (this.root_.key != key) {
            throw new Error('Key not found: ' + key as string);
        }
        var removed = this.root_;
        if (!this.root_.left) {
            this.root_ = this.root_.right;
        } else {
            var right = this.root_.right;
            this.root_ = this.root_.left;
            // Splay to make sure that the new root has an empty right child.
            this.splay_(key);
            // Insert the original right child as the right child of the new
            // root.
            this.root_.right = right;
        }
        return removed;
    }


    /**
     * Returns the node having the specified key or null if the tree doesn't contain
     * a node with the specified key.
     *
     * @param key Key to find in the tree.
     * @return {SplayTree.Node} Node having the specified key.
     */
    function find(key : number) : SplayTreeNode {
        if (this.isEmpty()) {
            return null;
        }
        this.splay_(key);
        return this.root_.key == key ? this.root_ : null;
    }


    /**
     * @return {SplayTree.Node} Node having the maximum key value.
     */
    function findMax(opt_startNode : SplayTreeNode) : SplayTreeNode {
        if (this.isEmpty()) {
            return null;
        }
        var current = opt_startNode == null? this.root_ : opt_startNode;
        while (current.right) {
            current = current.right;
        }
        return current;
    }


    /**
     * @return {SplayTree.Node} Node having the maximum key value that
     *     is less than the specified key value.
     */
    function findGreatestLessThan(key : number) : SplayTreeNode {
        if (this.isEmpty()) {
            return null;
        }
        // Splay on the key to move the node with the given key or the last
        // node on the search path to the top of the tree.
        this.splay_(key);
        // Now the result is either the root node or the greatest node in
        // the left subtree.
        if (this.root_.key < key) {
            return this.root_;
        } else if (this.root_.left) {
            return this.findMax(this.root_.left);
        } else {
            return null;
        }
    }


    /**
     * @return {Array<*>} An array containing all the keys of tree's nodes.
     */
    function exportKeys() : number[] {
        var result = [] : number[];
        if (!this.isEmpty()) {
            this.root_.traverse_(function(node) { result.push(node.key); });
        }
        return result;
    }


    /**
     * Perform the splay operation for the given key. Moves the node with
     * the given key to the top of the tree.  If no node has the given
     * key, the last node on the search path is moved to the top of the
     * tree. This is the simplified top-down splaying algorithm from:
     * "Self-adjusting Binary Search Trees" by Sleator and Tarjan
     *
     * @param key Key to splay the tree on.
     * @private
     */
    function splay_(key : number) : void {
        if (this.isEmpty()) {
            return;
        }
        // Create a dummy node.  The use of the dummy node is a bit
        // counter-intuitive: The right child of the dummy node will hold
        // the L tree of the algorithm.  The left child of the dummy node
        // will hold the R tree of the algorithm.  Using a dummy node, left
        // and right will always be nodes and we avoid special cases.
        var dummy, left, right;
        dummy = left = right = new SplayTreeNode(NaN, null);
        var current = this.root_;
        while (true) {
            if (key < current.key) {
                if (!current.left) {
                    break;
                }
                if (key < current.left.key) {
                    // Rotate right.
                    var tmp = current.left;
                    current.left = tmp.right;
                    tmp.right = current;
                    current = tmp;
                    if (!current.left) {
                        break;
                    }
                }
                // Link right.
                right.left = current;
                right = current;
                current = current.left;
            } else if (key > current.key) {
                if (!current.right) {
                    break;
                }
                if (key > current.right.key) {
                    // Rotate left.
                    var tmp = current.right;
                    current.right = tmp.left;
                    tmp.left = current;
                    current = tmp;
                    if (!current.right) {
                        break;
                    }
                }
                // Link left.
                left.right = current;
                left = current;
                current = current.right;
            } else {
                break;
            }
        }
        // Assemble.
        left.right = current.left;
        right.left = current.right;
        current.left = dummy.right;
        current.right = dummy.left;
        this.root_ = current;
    }
}


class SplayTreeNode {

    var key     : number;
    var value   : variant;

    // FIXME: wasabiz correct?

    /**
     * @type {SplayTree.Node}
     */
    var left = null : SplayTreeNode;


    /**
     * @type {SplayTree.Node}
     */
    var right = null : SplayTreeNode;


    /**
     * Constructs a Splay tree node.
     *
     * @param key Key.
     * @param value Value.
     */
    function constructor(key : number, value : variant) {
        this.key = key;
        this.value = value;
    }


    /**
     * Performs an ordered traversal of the subtree starting at
     * this SplayTree.Node.
     *
     * @param f Visitor function.
     * @private
     */
    function traverse_(f : (SplayTreeNode) -> void) : void {
        var current = this;
        while (current) {
            var left = current.left;
            if (left) left.traverse_(f);
            f(current);
            current = current.right;
        }
    }
}

// vim: set expandtab:
