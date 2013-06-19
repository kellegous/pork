// -*- mode: jsx; jsx-indent-level: 4; indent-tabs-mode: nil; -*-
// Copyright 2008 the V8 project authors. All rights reserved.
// Copyright 1996 John Maloney and Mario Wolczko.

// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program; if not, write to the Free Software
// Foundation, Inc., 59 Temple Place, Suite 330, Boston, MA  02111-1307  USA


// This implementation of the DeltaBlue benchmark is derived
// from the Smalltalk implementation by John Maloney and Mario
// Wolczko. Some parts have been translated directly, whereas
// others have been modified more aggresively to make it feel
// more like a JavaScript program.


import "./base.jsx";


class DeltaBlue {
    function constructor() {
        var DeltaBlue = new BenchmarkSuite('DeltaBlue', 66118, [
            new Benchmark('DeltaBlue', function() { Main.deltaBlue(); })
        ]);
    }
}


/**
 * A JavaScript implementation of the DeltaBlue constraint-solving
 * algorithm, as described in:
 *
 * "The DeltaBlue Algorithm: An Incremental Constraint Hierarchy Solver"
 *   Bjorn N. Freeman-Benson and John Maloney
 *   January 1990 Communications of the ACM,
 *   also available as University of Washington TR 89-08-06.
 *
 * Beware: this benchmark is written in a grotesque style where
 * the constraint model is built by side-effects from constructors.
 * I've kept it this way to avoid deviating too much from the original
 * implementation.
 */


/* --- O b j e c t   M o d e l --- */

class OrderedCollection.<T> {

    var elms : T[];

    function constructor() {
        this.elms = new Array.<T>();
    }

    function add(elm : T) : void {
        this.elms.push(elm);
    }

    function at(index : int) : T {
        return this.elms[index];
    }

    function size() : int {
        return this.elms.length;
    }

    function removeFirst() : T {
        return this.elms.pop();
    }

    function remove(elm : T) : void {
        var index = 0, skipped = 0;
        for (var i = 0; i < this.elms.length; i++) {
            var value = this.elms[i];
            if (value != elm) {
                this.elms[index] = value;
                index++;
            } else {
                skipped++;
            }
        }
        for (var i = 0; i < skipped; i++)
            this.elms.pop();
    }
}

/* --- *
 * S t r e n g t h
 * --- */

class Strength {

    // Strength constants.
    static const REQUIRED        = new Strength(0, "required");
    static const STONG_PREFERRED = new Strength(1, "strongPreferred");
    static const PREFERRED       = new Strength(2, "preferred");
    static const STRONG_DEFAULT  = new Strength(3, "strongDefault");
    static const NORMAL          = new Strength(4, "normal");
    static const WEAK_DEFAULT    = new Strength(5, "weakDefault");
    static const WEAKEST         = new Strength(6, "weakest");

    var strengthValue : int;
    var name          : string;

    /**
     * Strengths are used to measure the relative importance of constraints.
     * New strengths may be inserted in the strength hierarchy without
     * disrupting current constraints.  Strengths cannot be created outside
     * this class, so pointer comparison can be used for value comparison.
     */
    function constructor(strengthValue : int, name : string) {
        this.strengthValue = strengthValue;
        this.name = name;
    }

    static function stronger(s1 : Strength, s2 : Strength) : boolean {
        return s1.strengthValue < s2.strengthValue;
    }

    static function weaker(s1 : Strength, s2 : Strength) : boolean {
        return s1.strengthValue > s2.strengthValue;
    }

    static function weakestOf(s1 : Strength, s2 : Strength) : Strength {
        return Strength.weaker(s1, s2) ? s1 : s2;
    }

    static function strongest(s1 : Strength, s2 : Strength) : Strength {
        return Strength.stronger(s1, s2) ? s1 : s2;
    }

    function nextWeaker() : Strength {
        switch (this.strengthValue) {
        case 0: return Strength.WEAKEST;
        case 1: return Strength.WEAK_DEFAULT;
        case 2: return Strength.NORMAL;
        case 3: return Strength.STRONG_DEFAULT;
        case 4: return Strength.PREFERRED;
        case 5: return Strength.REQUIRED;
        default: return null;
        }
    }
}

/* --- *
 * C o n s t r a i n t
 * --- */

abstract class Constraint {

    var strength : Strength;

    /**
     * An abstract class representing a system-maintainable relationship
     * (or "constraint") between a set of variables. A constraint supplies
     * a strength instance variable; concrete subclasses provide a means
     * of storing the constrained variables and other information required
     * to represent a constraint.
     */
    function constructor(strength : Strength) {
        this.strength = strength;
    }

    abstract function addToGraph() : void;
    abstract function removeFromGraph() : void;

    abstract function execute() : void;

    abstract function chooseMethod(mark : int) : void;
    abstract function isSatisfied() : boolean;
    abstract function markInputs(mark : int) : void;
    abstract function inputsKnown(mark : int) : boolean;
    abstract function output() : Variable;
    abstract function markUnsatisfied() : void;
    abstract function recalculate() : void;

    /**
     * Activate this constraint and attempt to satisfy it.
     */
    function addConstraint() : void {
        this.addToGraph();
        Main.planner.incrementalAdd(this);
    }

    /**
     * Attempt to find a way to enforce this constraint. If successful,
     * record the solution, perhaps modifying the current dataflow
     * graph. Answer the constraint that this constraint overrides, if
     * there is one, or nil, if there isn't.
     * Assume: I am not already satisfied.
     */
    function satisfy(mark : int) : Constraint {
        this.chooseMethod(mark);
        if (!this.isSatisfied()) {
            if (this.strength == Strength.REQUIRED)
                log "Could not satisfy a required constraint!";
            return null;
        }
        this.markInputs(mark);
        var out = this.output();
        var overridden = out.determinedBy;
        if (overridden != null) overridden.markUnsatisfied();
        out.determinedBy = this;
        if (!Main.planner.addPropagate(this, mark))
            log "Cycle encountered";
        out.mark = mark;
        return overridden;
    }

    function destroyConstraint() : void {
        if (this.isSatisfied()) Main.planner.incrementalRemove(this);
        else this.removeFromGraph();
    }

    /**
     * Normal constraints are not input constraints.  An input constraint
     * is one that depends on external state, such as the mouse, the
     * keybord, a clock, or some arbitraty piece of imperative code.
     */
    function isInput() : boolean {
        return false;
    }
}

/* --- *
 * U n a r y   C o n s t r a i n t
 * --- */

abstract class UnaryConstraint extends Constraint {

    var myOutput : Variable;
    var satisfied : boolean;

    /**
     * Abstract superclass for constraints having a single possible output
     * variable.
     */
    function constructor(v : Variable, strength : Strength) {
        super(strength);
        this.myOutput = v;
        this.satisfied = false;
        this.addConstraint();
    }

    /**
     * Adds this constraint to the constraint graph
     */
    override function addToGraph() : void {
        this.myOutput.addConstraint(this);
        this.satisfied = false;
    }

    /**
     * Decides if this constraint can be satisfied and records that
     * decision.
     */
    override function chooseMethod(mark : int) : void {
        this.satisfied = (this.myOutput.mark != mark)
        && Strength.stronger(this.strength, this.myOutput.walkStrength);
    }

    /**
     * Returns true if this constraint is satisfied in the current solution.
     */
    override function isSatisfied() : boolean {
        return this.satisfied;
    }

    override function markInputs(mark : int) : void {
        // has no inputs
    }

    /**
     * Returns the current output variable.
     */
    override function output() : Variable {
        return this.myOutput;
    }

    /**
     * Calculate the walkabout strength, the stay flag, and, if it is
     * 'stay', the value for the current output of this constraint. Assume
     * this constraint is satisfied.
     */
    override function recalculate() : void {
        this.myOutput.walkStrength = this.strength;
        this.myOutput.stay = !this.isInput();
        if (this.myOutput.stay) this.execute(); // Stay optimization
    }

    /**
     * Records that this constraint is unsatisfied
     */
    override function markUnsatisfied() : void {
        this.satisfied = false;
    }

    override function inputsKnown(mark : int) : boolean {
        return true;
    }

    override function removeFromGraph() : void {
        if (this.myOutput != null) this.myOutput.removeConstraint(this);
        this.satisfied = false;
    }
}

/* --- *
 * S t a y   C o n s t r a i n t
 * --- */

class StayConstraint extends UnaryConstraint {

    /**
     * Variables that should, with some level of preference, stay the same.
     * Planners may exploit the fact that instances, if satisfied, will not
     * change their output during plan execution.  This is called "stay
     * optimization".
     */
    function constructor(v : Variable, str : Strength) {
        super(v, str);
    }

    override function execute() : void {
        // Stay constraints do nothing
    }
}

/* --- *
 * E d i t   C o n s t r a i n t
 * --- */

class EditConstraint extends UnaryConstraint {

    /**
     * A unary input constraint used to mark a variable that the client
     * wishes to change.
     */
    function constructor(v : Variable, str : Strength) {
        super(v, str);
    }

    /**
     * Edits indicate that a variable is to be changed by imperative code.
     */
    override function isInput() : boolean {
        return true;
    }

    override function execute() : void {
        // Edit constraints do nothing
    }
}

/* --- *
 * B i n a r y   C o n s t r a i n t
 * --- */

class Direction {
    static const NONE = 0;
    static const FORWARD = 1;
    static const BACKWARD = -1;
}

abstract class BinaryConstraint extends Constraint {

    var v1 : Variable;
    var v2 : Variable;
    var direction : int;

    /**
     * Abstract superclass for constraints having two possible output
     * variables.
     */
    function constructor(var1 : Variable, var2 : Variable, strength : Strength) {
        super(strength);
        this.v1 = var1;
        this.v2 = var2;
        this.direction = Direction.NONE;
        this.addConstraint();
    }

    /**
     * Protected constructor: this will not call a member function addConstraint;
     * only calls super constructor.
     */
    function constructor(strength : Strength) {
        super(strength);
    }

    /**
     * Decides if this constraint can be satisfied and which way it
     * should flow based on the relative strength of the variables related,
     * and record that decision.
     */
    override function chooseMethod(mark : int) : void {
        if (this.v1.mark == mark) {
            this.direction = (this.v2.mark != mark && Strength.stronger(this.strength, this.v2.walkStrength))
            ? Direction.FORWARD
            : Direction.NONE;
        }
        if (this.v2.mark == mark) {
            this.direction = (this.v1.mark != mark && Strength.stronger(this.strength, this.v1.walkStrength))
            ? Direction.BACKWARD
            : Direction.NONE;
        }
        if (Strength.weaker(this.v1.walkStrength, this.v2.walkStrength)) {
            this.direction = Strength.stronger(this.strength, this.v1.walkStrength)
            ? Direction.BACKWARD
            : Direction.NONE;
        } else {
            this.direction = Strength.stronger(this.strength, this.v2.walkStrength)
            ? Direction.FORWARD
            : Direction.BACKWARD;
        }
    }

    /**
     * Add this constraint to the constraint graph
     */
    override function addToGraph() : void {
        this.v1.addConstraint(this);
        this.v2.addConstraint(this);
        this.direction = Direction.NONE;
    }

    /**
     * Answer true if this constraint is satisfied in the current solution.
     */
    override function isSatisfied() : boolean {
        return this.direction != Direction.NONE;
    }

    /**
     * Mark the input variable with the given mark.
     */
    override function markInputs(mark : int) : void {
        this.input().mark = mark;
    }

    /**
     * Returns the current input variable
     */
    function input() : Variable {
        return (this.direction == Direction.FORWARD) ? this.v1 : this.v2;
    }

    /**
     * Returns the current output variable
     */
    override function output() : Variable {
        return (this.direction == Direction.FORWARD) ? this.v2 : this.v1;
    }

    /**
     * Calculate the walkabout strength, the stay flag, and, if it is
     * 'stay', the value for the current output of this
     * constraint. Assume this constraint is satisfied.
     */
    override function recalculate() : void {
        var ihn = this.input(), out = this.output();
        out.walkStrength = Strength.weakestOf(this.strength, ihn.walkStrength);
        out.stay = ihn.stay;
        if (out.stay) this.execute();
    }

    /**
     * Record the fact that this constraint is unsatisfied.
     */
    override function markUnsatisfied() : void {
        this.direction = Direction.NONE;
    }

    override function inputsKnown(mark : int) : boolean {
        var i = this.input();
        return i.mark == mark || i.stay || i.determinedBy == null;
    }

    override function removeFromGraph() : void {
        if (this.v1 != null) this.v1.removeConstraint(this);
        if (this.v2 != null) this.v2.removeConstraint(this);
        this.direction = Direction.NONE;
    }
}

/* --- *
 * S c a l e   C o n s t r a i n t
 * --- */

class ScaleConstraint extends BinaryConstraint {

    var scale                : Variable;
    var offset                : Variable;

    /**
     * Relates two variables by the linear scaling relationship: "v2 =
     * (v1 * scale) + offset". Either v1 or v2 may be changed to maintain
     * this relationship but the scale factor and offset are considered
     * read-only.
     */
    function constructor(src : Variable, scale : Variable, offset : Variable, dest : Variable, strength : Strength) {
        super(strength);
        this.v1 = src;
        this.v2 = dest;
        this.direction = Direction.NONE;
        this.scale = scale;
        this.offset = offset;
        this.addConstraint();
    }

    /**
     * Adds this constraint to the constraint graph.
     */
    override function addToGraph() : void {
        super.addToGraph();
        this.scale.addConstraint(this);
        this.offset.addConstraint(this);
    }

    override function removeFromGraph() : void {
        super.removeFromGraph();
        if (this.scale != null) this.scale.removeConstraint(this);
        if (this.offset != null) this.offset.removeConstraint(this);
    }

    override function markInputs(mark : int) : void {
        super.markInputs(mark);
        this.scale.mark = this.offset.mark = mark;
    }

    /**
     * Enforce this constraint. Assume that it is satisfied.
     */
    override function execute() : void {
        if (this.direction == Direction.FORWARD) {
            this.v2.value = this.v1.value * this.scale.value + this.offset.value;
        } else {
            this.v1.value = (this.v2.value - this.offset.value) / this.scale.value;
        }
    }

    /**
     * Calculate the walkabout strength, the stay flag, and, if it is
     * 'stay', the value for the current output of this constraint. Assume
     * this constraint is satisfied.
     */
    override function recalculate() : void {
        var ihn = this.input(), out = this.output();
        out.walkStrength = Strength.weakestOf(this.strength, ihn.walkStrength);
        out.stay = ihn.stay && this.scale.stay && this.offset.stay;
        if (out.stay) this.execute();
    }
}

/* --- *
 * E q u a l i t  y   C o n s t r a i n t
 * --- */

class EqualityConstraint extends BinaryConstraint {

    /**
     * Constrains two variables to have the same value.
     */
    function constructor(var1 : Variable, var2 : Variable, strength : Strength) {
        super(var1, var2, strength);
    }

    /**
     * Enforce this constraint. Assume that it is satisfied.
     */
    override function execute() : void {
        this.output().value = this.input().value;
    }
}

/* --- *
 * V a r i a b l e
 * --- */

class Variable {

    var value                : number;
    var constraints        : OrderedCollection.<Constraint>;
    var determinedBy        : Constraint;
    var mark                : int;
    var walkStrength        : Strength;
    var stay                : boolean;
    var name                : string;

    /**
     * A constrained variable. In addition to its value, it maintain the
     * structure of the constraint graph, the current dataflow graph, and
     * various parameters of interest to the DeltaBlue incremental
     * constraint solver.
     **/
    function constructor(name : string) {
        this(name, 0);
    }

    function constructor(name : string, initialValue : number) {
        this.value = initialValue;
        this.constraints = new OrderedCollection.<Constraint>();
        this.determinedBy = null;
        this.mark = 0;
        this.walkStrength = Strength.WEAKEST;
        this.stay = true;
        this.name = name;
    }

    /**
     * Add the given constraint to the set of all constraints that refer
     * this variable.
     */
    function addConstraint(c : Constraint) : void {
        this.constraints.add(c);
    }

    /**
     * Removes all traces of c from this variable.
     */
    function removeConstraint(c : Constraint) : void {
        this.constraints.remove(c);
        if (this.determinedBy == c) this.determinedBy = null;
    }
}

/* --- *
 * P l a n n e r
 * --- */

class Planner {

    var currentMark : int;

    /**
     * The DeltaBlue planner
     */
    function constructor() {
        this.currentMark = 0;
    }

    /**
     * Attempt to satisfy the given constraint and, if successful,
     * incrementally update the dataflow graph.  Details: If satifying
     * the constraint is successful, it may override a weaker constraint
     * on its output. The algorithm attempts to resatisfy that
     * constraint using some other method. This process is repeated
     * until either a) it reaches a variable that was not previously
     * determined by any constraint or b) it reaches a constraint that
     * is too weak to be satisfied using any of its methods. The
     * variables of constraints that have been processed are marked with
     * a unique mark value so that we know where we've been. This allows
     * the algorithm to avoid getting into an infinite loop even if the
     * constraint graph has an inadvertent cycle.
     */
    function incrementalAdd(c : Constraint) : void {
        var mark = this.newMark();
        var overridden = c.satisfy(mark);
        while (overridden != null)
            overridden = overridden.satisfy(mark);
    }

    /**
     * Entry point for retracting a constraint. Remove the given
     * constraint and incrementally update the dataflow graph.
     * Details: Retracting the given constraint may allow some currently
     * unsatisfiable downstream constraint to be satisfied. We therefore collect
     * a list of unsatisfied downstream constraints and attempt to
     * satisfy each one in turn. This list is traversed by constraint
     * strength, strongest first, as a heuristic for avoiding
     * unnecessarily adding and then overriding weak constraints.
     * Assume: c is satisfied.
     */
    function incrementalRemove(c : Constraint) : void {
        var out = c.output();
        c.markUnsatisfied();
        c.removeFromGraph();
        var unsatisfied = this.removePropagateFrom(out);
        var strength = Strength.REQUIRED;
        do {
            for (var i = 0; i < unsatisfied.size(); i++) {
                var u = unsatisfied.at(i);
                if (u.strength == strength)
                    this.incrementalAdd(u);
            }
            strength = strength.nextWeaker();
        } while (strength != Strength.WEAKEST);
    }

    /**
     * Select a previously unused mark value.
     */
    function newMark() : int {
        return ++this.currentMark;
    }

    /**
     * Extract a plan for resatisfaction starting from the given source
     * constraints, usually a set of input constraints. This method
     * assumes that stay optimization is desired; the plan will contain
     * only constraints whose output variables are not stay. Constraints
     * that do no computation, such as stay and edit constraints, are
     * not included in the plan.
     * Details: The outputs of a constraint are marked when it is added
     * to the plan under construction. A constraint may be appended to
     * the plan when all its input variables are known. A variable is
     * known if either a) the variable is marked (indicating that has
     * been computed by a constraint appearing earlier in the plan), b)
     * the variable is 'stay' (i.e. it is a constant at plan execution
     * time), or c) the variable is not determined by any
     * constraint. The last provision is for past states of history
     * variables, which are not stay but which are also not computed by
     * any constraint.
     * Assume: sources are all satisfied.
     */
    function makePlan(sources : OrderedCollection.<Constraint>) : Plan {
        var mark = this.newMark();
        var plan = new Plan();
        var todo = sources;
        while (todo.size() > 0) {
            var c = todo.removeFirst();
            if (c.output().mark != mark && c.inputsKnown(mark)) {
                plan.addConstraint(c);
                c.output().mark = mark;
                this.addConstraintsConsumingTo(c.output(), todo);
            }
        }
        return plan;
    }

    /**
     * Extract a plan for resatisfying starting from the output of the
     * given constraints, usually a set of input constraints.
     */
    function extractPlanFromConstraints(constraints : OrderedCollection.<Constraint>) : Plan {
        var sources = new OrderedCollection.<Constraint>();
        for (var i = 0; i < constraints.size(); i++) {
            var c = constraints.at(i);
            if (c.isInput() && c.isSatisfied())
                // not in plan already and eligible for inclusion
                sources.add(c);
        }
        return this.makePlan(sources);
    }

    /**
     * Recompute the walkabout strengths and stay flags of all variables
     * downstream of the given constraint and recompute the actual
     * values of all variables whose stay flag is true. If a cycle is
     * detected, remove the given constraint and answer
     * false. Otherwise, answer true.
     * Details: Cycles are detected when a marked variable is
     * encountered downstream of the given constraint. The sender is
     * assumed to have marked the inputs of the given constraint with
     * the given mark. Thus, encountering a marked node downstream of
     * the output constraint means that there is a path from the
     * constraint's output to one of its inputs.
     */
    function addPropagate(c : Constraint, mark : int) : boolean {
        var todo = new OrderedCollection.<Constraint>();
        todo.add(c);
        while (todo.size() > 0) {
            var d = todo.removeFirst();
            if (d.output().mark == mark) {
                this.incrementalRemove(c);
                return false;
            }
            d.recalculate();
            this.addConstraintsConsumingTo(d.output(), todo);
        }
        return true;
    }


    /**
     * Update the walkabout strengths and stay flags of all variables
     * downstream of the given constraint. Answer a collection of
     * unsatisfied constraints sorted in order of decreasing strength.
     */
    function removePropagateFrom(out : Variable) : OrderedCollection.<Constraint> {
        out.determinedBy = null;
        out.walkStrength = Strength.WEAKEST;
        out.stay = true;
        var unsatisfied = new OrderedCollection.<Constraint>();
        var todo = new OrderedCollection.<Variable>();
        todo.add(out);
        while (todo.size() > 0) {
            var v = todo.removeFirst();
            for (var i = 0; i < v.constraints.size(); i++) {
                var c = v.constraints.at(i);
                if (!c.isSatisfied())
                    unsatisfied.add(c);
            }
            var determining = v.determinedBy;
            for (var i = 0; i < v.constraints.size(); i++) {
                var next = v.constraints.at(i);
                if (next != determining && next.isSatisfied()) {
                    next.recalculate();
                    todo.add(next.output());
                }
            }
        }
        return unsatisfied;
    }

    function addConstraintsConsumingTo(v : Variable, coll : OrderedCollection.<Constraint>) : void {
        var determining = v.determinedBy;
        var cc = v.constraints;
        for (var i = 0; i < cc.size(); i++) {
            var c = cc.at(i);
            if (c != determining && c.isSatisfied())
                coll.add(c);
        }
    }
}

/* --- *
 * P l a n
 * --- */

class Plan {

    var v : OrderedCollection.<Constraint>;

    /**
     * A Plan is an ordered list of constraints to be executed in sequence
     * to resatisfy all currently satisfiable constraints in the face of
     * one or more changing inputs.
     */
    function constructor() {
        this.v = new OrderedCollection.<Constraint>();
    }

    function addConstraint(c : Constraint) : void {
        this.v.add(c);
    }

    function size() : int {
        return this.v.size();
    }

    function constraintAt(index : int) : Constraint {
        return this.v.at(index);
    }

    function execute() : void {
        for (var i = 0; i < this.size(); i++) {
            var c = this.constraintAt(i);
            c.execute();
        }
    }
}

/* --- *
 * M a i n
 * --- */

class Main {

    // Global variable holding the current planner.
    static var planner = null : Planner;

    /**
     * This is the standard DeltaBlue benchmark. A long chain of equality
     * constraints is constructed with a stay constraint on one end. An
     * edit constraint is then added to the opposite end and the time is
     * measured for adding and removing this constraint, and extracting
     * and executing a constraint satisfaction plan. There are two cases.
     * In case 1, the added constraint is stronger than the stay
     * constraint and values must propagate down the entire length of the
     * chain. In case 2, the added constraint is weaker than the stay
     * constraint so it cannot be accomodated. The cost in this case is,
     * of course, very low. Typical situations lie somewhere between these
     * two extremes.
     */
    static function chainTest(n : int) : void {
        Main.planner = new Planner();
        var prev = null : Variable;
        var first = null : Variable;
        var last = null : Variable;

        // Build chain of n equality constraints
        for (var i = 0; i <= n; i++) {
            var name = "v" + i as string;
            var v = new Variable(name);
            if (prev != null)
                new EqualityConstraint(prev, v, Strength.REQUIRED);
            if (i == 0) first = v;
            if (i == n) last = v;
            prev = v;
        }

        new StayConstraint(last, Strength.STRONG_DEFAULT);
        var edit = new EditConstraint(first, Strength.PREFERRED);
        var edits = new OrderedCollection.<Constraint>();
        edits.add(edit);
        var plan = Main.planner.extractPlanFromConstraints(edits);
        for (var i = 0; i < 100; i++) {
            first.value = i;
            plan.execute();
            if (last.value != i)
                log "Chain test failed.";
        }
    }

    /**
     * This test constructs a two sets of variables related to each
     * other by a simple linear transformation (scale and offset). The
     * time is measured to change a variable on either side of the
     * mapping and to change the scale and offset factors.
     */
    static function projectionTest(n : int) : void {
        Main.planner = new Planner();
        var scale = new Variable("scale", 10);
        var offset = new Variable("offset", 1000);
        var src = null : Variable;
        var dst = null : Variable;

        var dests = new OrderedCollection.<Variable>();
        for (var i = 0; i < n; i++) {
            src = new Variable("src" + i as string, i);
            dst = new Variable("dst" + i as string, i);
            dests.add(dst);
            new StayConstraint(src, Strength.NORMAL);
            new ScaleConstraint(src, scale, offset, dst, Strength.REQUIRED);
        }

        Main.change(src, 17);
        if (dst.value != 1170) log "Projection 1 failed";
        Main.change(dst, 1050);
        if (src.value != 5) log "Projection 2 failed";
        Main.change(scale, 5);
        for (var i = 0; i < n - 1; i++) {
            if (dests.at(i).value != i * 5 + 1000)
                log "Projection 3 failed";
        }
        Main.change(offset, 2000);
        for (var i = 0; i < n - 1; i++) {
            if (dests.at(i).value != i * 5 + 2000)
                log "Projection 4 failed";
        }
    }

    static function change(v : Variable, newValue : number) : void {
        var edit = new EditConstraint(v, Strength.PREFERRED);
        var edits = new OrderedCollection.<Constraint>();
        edits.add(edit);
        var plan = Main.planner.extractPlanFromConstraints(edits);
        for (var i = 0; i < 10; i++) {
            v.value = newValue;
            plan.execute();
        }
        edit.destroyConstraint();
    }

    static function deltaBlue() : void {
        Main.chainTest(100);
        Main.projectionTest(100);
    }
}
// vim: set expandtab:
