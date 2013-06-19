// -*- mode: jsx; jsx-indent-level: 4; indent-tabs-mode: nil; -*-
// Copyright 2006-2008 the V8 project authors. All rights reserved.
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


import "./base.jsx";


// This is a JavaScript implementation of the Richards
// benchmark from:
//
//    http://www.cl.cam.ac.uk/~mr10/Bench.html
//
// The benchmark was originally implemented in BCPL by
// Martin Richards.


class Richards {


    static const COUNT = 1000;

    /**
     * These two constants specify how many times a packet is queued and
     * how many times a task is put on hold in a correct run of richards.
     * They don't have any meaning a such but are characteristic of a
     * correct run so if the actual queue or hold count is different from
     * the expected there must be a bug in the implementation.
     **/
    static const EXPECTED_QUEUE_COUNT = 2322;
    static const EXPECTED_HOLD_COUNT = 928;


    function constructor() {
        var richards = new BenchmarkSuite('Richards', 35302, [
            new Benchmark("Richards", function() { this.runRichards(); })
            ]);
    }


    /**
     * The Richards benchmark simulates the task dispatcher of an
     * operating system.
     **/
    function runRichards() : void {
        var scheduler = new Scheduler();
        scheduler.addIdleTask(Scheduler.ID_IDLE, 0, null, Richards.COUNT);

        var queue = new Packet(null, Scheduler.ID_WORKER, Scheduler.KIND_WORK);
        queue = new Packet(queue,  Scheduler.ID_WORKER, Scheduler.KIND_WORK);
        scheduler.addWorkerTask(Scheduler.ID_WORKER, 1000, queue);

        queue = new Packet(null, Scheduler.ID_DEVICE_A, Scheduler.KIND_DEVICE);
        queue = new Packet(queue,  Scheduler.ID_DEVICE_A, Scheduler.KIND_DEVICE);
        queue = new Packet(queue,  Scheduler.ID_DEVICE_A, Scheduler.KIND_DEVICE);
        scheduler.addHandlerTask(Scheduler.ID_HANDLER_A, 2000, queue);

        queue = new Packet(null, Scheduler.ID_DEVICE_B, Scheduler.KIND_DEVICE);
        queue = new Packet(queue,  Scheduler.ID_DEVICE_B, Scheduler.KIND_DEVICE);
        queue = new Packet(queue,  Scheduler.ID_DEVICE_B, Scheduler.KIND_DEVICE);
        scheduler.addHandlerTask(Scheduler.ID_HANDLER_B, 3000, queue);

        scheduler.addDeviceTask(Scheduler.ID_DEVICE_A, 4000, null);

        scheduler.addDeviceTask(Scheduler.ID_DEVICE_B, 5000, null);

        scheduler.schedule();

        if (scheduler.queueCount != Richards.EXPECTED_QUEUE_COUNT ||
            scheduler.holdCount != Richards.EXPECTED_HOLD_COUNT) {
                var msg =
                "Error during execution: queueCount = " + scheduler.queueCount as string +
                ", holdCount = " + scheduler.holdCount as string + ".";
                throw new Error(msg);
            }
    }
}

class Scheduler {

    static const ID_IDLE       = 0;
    static const ID_WORKER     = 1;
    static const ID_HANDLER_A  = 2;
    static const ID_HANDLER_B  = 3;
    static const ID_DEVICE_A   = 4;
    static const ID_DEVICE_B   = 5;
    static const NUMBER_OF_IDS = 6;

    static const KIND_DEVICE   = 0;
    static const KIND_WORK     = 1;


    var queueCount      : int;
    var holdCount       : int;
    var blocks          : TaskControlBlock[];
    var list            : TaskControlBlock;
    var currentTcb      : TaskControlBlock;
    var currentId       : Nullable.<int>;

    /**
     * A scheduler can be used to schedule a set of tasks based on their relative
     * priorities.  Scheduling is done by maintaining a list of task control blocks
     * which holds tasks and the data queue they are processing.
     * @constructor
     */
    function constructor() {
        this.queueCount = 0;
        this.holdCount = 0;
        this.blocks = new Array.<TaskControlBlock>(Scheduler.NUMBER_OF_IDS);
        this.list = null;
        this.currentTcb = null;
        this.currentId = null;
    }

    /**
     * Add an idle task to this scheduler.
     * @param id the identity of the task
     * @param priority the task's priority
     * @param queue the queue of work to be processed by the task
     * @param count the number of times to schedule the task
     */
    function addIdleTask(id : int, priority : int, queue : Packet, count : int) : void {
        this.addRunningTask(id, priority, queue, new IdleTask(this, 1, count));
    }

    /**
     * Add a work task to this scheduler.
     * @param id the identity of the task
     * @param priority the task's priority
     * @param queue the queue of work to be processed by the task
     */
    function addWorkerTask(id : int, priority : int , queue : Packet) : void {
        this.addTask(id, priority, queue, new WorkerTask(this, Scheduler.ID_HANDLER_A, 0));
    }

    /**
     * Add a handler task to this scheduler.
     * @param id the identity of the task
     * @param priority the task's priority
     * @param queue the queue of work to be processed by the task
     */
    function addHandlerTask(id : int, priority : int, queue : Packet) : void {
        this.addTask(id, priority, queue, new HandlerTask(this));
    }

    /**
     * Add a handler task to this scheduler.
     * @param id the identity of the task
     * @param priority the task's priority
     * @param queue the queue of work to be processed by the task
     */
    function addDeviceTask(id : int, priority : int, queue : Packet) : void {
        this.addTask(id, priority, queue, new DeviceTask(this));
    }

    /**
     * Add the specified task and mark it as running.
     * @param id the identity of the task
     * @param priority the task's priority
     * @param queue the queue of work to be processed by the task
     * @param task the task to add
     */
    function addRunningTask(id : int, priority : int, queue : Packet, task : Task) : void {
        this.addTask(id, priority, queue, task);
        this.currentTcb.setRunning();
    }

    /**
     * Add the specified task to this scheduler.
     * @param id the identity of the task
     * @param priority the task's priority
     * @param queue the queue of work to be processed by the task
     * @param task the task to add
     */
    function addTask(id : int, priority : int, queue : Packet, task : Task) : void  {
        this.currentTcb = new TaskControlBlock(this.list, id, priority, queue, task);
        this.list = this.currentTcb;
        this.blocks[id] = this.currentTcb;
    }

    /**
     * Execute the tasks managed by this scheduler.
     */
    function schedule() : void {
        this.currentTcb = this.list;
        while (this.currentTcb != null) {
            if (this.currentTcb.isHeldOrSuspended()) {
                this.currentTcb = this.currentTcb.link;
            } else {
                this.currentId = this.currentTcb.id;
                this.currentTcb = this.currentTcb.run();
            }
        }
    }

    /**
     * Release a task that is currently blocked and return the next block to run.
     * @param id the id of the task to suspend
     */
    function release(id : int) : TaskControlBlock {
        var tcb = this.blocks[id];
        if (tcb == null) return tcb;
        tcb.markAsNotHeld();
        if (tcb.priority > this.currentTcb.priority) {
            return tcb;
        } else {
            return this.currentTcb;
        }
    }

    /**
     * Block the currently executing task and return the next task control block
     * to run.  The blocked task will not be made runnable until it is explicitly
     * released, even if new work is added to it.
     */
    function holdCurrent() : TaskControlBlock {
        this.holdCount++;
        this.currentTcb.markAsHeld();
        return this.currentTcb.link;
    }

    /**
     * Suspend the currently executing task and return the next task control block
     * to run.  If new work is added to the suspended task it will be made runnable.
     */
    function suspendCurrent() : TaskControlBlock {
        this.currentTcb.markAsSuspended();
        return this.currentTcb;
    }

    /**
     * Add the specified packet to the end of the worklist used by the task
     * associated with the packet and make the task runnable if it is currently
     * suspended.
     * @param packet the packet to add
     */
    function queue(packet : Packet) : TaskControlBlock {
        var t = this.blocks[packet.id];
        if (t == null) return t;
        this.queueCount++;
        packet.link = null;
        packet.id = this.currentId;
        return t.checkPriorityAdd(this.currentTcb, packet);
    }
}

class TaskControlBlock {

    /**
     * The task is running and is currently scheduled.
     */
    static const STATE_RUNNING = 0;

    /**
     * The task has packets left to process.
     */
    static const STATE_RUNNABLE = 1;

    /**
     * The task is not currently running.  The task is not blocked as such and may
     * be started by the scheduler.
     */
    static const STATE_SUSPENDED = 2;

    /**
     * The task is blocked and cannot be run until it is explicitly released.
     */
    static const STATE_HELD = 4;

    static const STATE_SUSPENDED_RUNNABLE = TaskControlBlock.STATE_SUSPENDED | TaskControlBlock.STATE_RUNNABLE;
    static const STATE_NOT_HELD = ~TaskControlBlock.STATE_HELD;


    var link            : TaskControlBlock;
    var id              : int;
    var priority        : int;
    var queue           : Packet;
    var task            : Task;
    var state           : int;

    /**
     * A task control block manages a task and the queue of work packages associated
     * with it.
     * @param link the preceding block in the linked block list
     * @param id the id of this block
     * @param priority the priority of this block
     * @param queue the queue of packages to be processed by the task
     * @param task the task
     * @constructor
     */
    function constructor(link : TaskControlBlock, id : int, priority : int, queue : Packet, task : Task) {
        this.link = link;
        this.id = id;
        this.priority = priority;
        this.queue = queue;
        this.task = task;
        if (queue == null) {
            this.state = TaskControlBlock.STATE_SUSPENDED;
        } else {
            this.state = TaskControlBlock.STATE_SUSPENDED_RUNNABLE;
        }
    }

    function setRunning() : void {
        this.state = TaskControlBlock.STATE_RUNNING;
    }

    function markAsNotHeld() : void {
        this.state = this.state & TaskControlBlock.STATE_NOT_HELD;
    }

    function markAsHeld() : void {
        this.state = this.state | TaskControlBlock.STATE_HELD;
    }

    function isHeldOrSuspended() : boolean {
        return (this.state & TaskControlBlock.STATE_HELD) != 0 || (this.state == TaskControlBlock.STATE_SUSPENDED);
    }

    function markAsSuspended() : void {
        this.state = this.state | TaskControlBlock.STATE_SUSPENDED;
    }

    function markAsRunnable() : void {
        this.state = this.state | TaskControlBlock.STATE_RUNNABLE;
    }

    /**
     * Runs this task, if it is ready to be run, and returns the next task to run.
     */
    function run() : TaskControlBlock {
        var packet;
        if (this.state == TaskControlBlock.STATE_SUSPENDED_RUNNABLE) {
            packet = this.queue;
            this.queue = packet.link;
            if (this.queue == null) {
                this.state = TaskControlBlock.STATE_RUNNING;
            } else {
                this.state = TaskControlBlock.STATE_RUNNABLE;
            }
        } else {
            packet = null;
        }
        return this.task.run(packet);
    }

    /**
     * Adds a packet to the worklist of this block's task, marks this as runnable if
     * necessary, and returns the next runnable object to run (the one
     * with the highest priority).
     */
    function checkPriorityAdd(task : TaskControlBlock, packet : Packet) : TaskControlBlock {
        if (this.queue == null) {
            this.queue = packet;
            this.markAsRunnable();
            if (this.priority > task.priority) return this;
        } else {
            this.queue = packet.addTo(this.queue);
        }
        return task;
    }

    override function toString() : string {
        return "tcb { " + this.task.toString() + "@" + this.state.toString() + " }";
    }
}


abstract class Task {

    var scheduler : Scheduler;

    function constructor(scheduler : Scheduler) {
        this.scheduler = scheduler;
    }

    abstract function run(packet : Packet) : TaskControlBlock;
}

class IdleTask extends Task {

    var v1      : int;
    var count   : int;

    /**
     * An idle task doesn't do any work itself but cycles control between the two
     * device tasks.
     * @param scheduler the scheduler that manages this task
     * @param v1 a seed value that controls how the device tasks are scheduled
     * @param count the number of times this task should be scheduled
     * @constructor
     */
    function constructor(scheduler : Scheduler, v1 : int, count : int) {
        super(scheduler);
        this.v1 = v1;
        this.count = count;
    }

    override function run(packet : Packet) : TaskControlBlock {
        this.count--;
        if (this.count == 0) return this.scheduler.holdCurrent();
        if ((this.v1 & 1) == 0) {
            this.v1 = this.v1 >> 1;
            return this.scheduler.release(Scheduler.ID_DEVICE_A);
        } else {
            this.v1 = (this.v1 >> 1) ^ 0xD008;
            return this.scheduler.release(Scheduler.ID_DEVICE_B);
        }
    }

    override function toString() : string {
        return "IdleTask";
    }
}

class DeviceTask extends Task {

    var v1 : Packet;

    /**
     * A task that suspends itself after each time it has been run to simulate
     * waiting for data from an external device.
     * @param scheduler the scheduler that manages this task
     * @constructor
     */
    function constructor(scheduler : Scheduler) {
        super(scheduler);
        this.v1 = null;
    }

    override function run(packet : Packet) : TaskControlBlock {
        if (packet == null) {
            if (this.v1 == null) return this.scheduler.suspendCurrent();
            var v = this.v1;
            this.v1 = null;
            return this.scheduler.queue(v);
        } else {
            this.v1 = packet;
            return this.scheduler.holdCurrent();
        }
    }

    override function toString() : string {
        return "DeviceTask";
    }
}

class WorkerTask extends Task {

    var v1 : int;
    var v2 : int;

    /**
     * A task that manipulates work packets.
     * @param scheduler the scheduler that manages this task
     * @param v1 a seed used to specify how work packets are manipulated
     * @param v2 another seed used to specify how work packets are manipulated
     * @constructor
     */
    function constructor(scheduler : Scheduler, v1 : int, v2 : int) {
        super(scheduler);
        this.v1 = v1;
        this.v2 = v2;
    }

    override function run(packet : Packet) : TaskControlBlock {
        if (packet == null) {
            return this.scheduler.suspendCurrent();
        } else {
            if (this.v1 == Scheduler.ID_HANDLER_A) {
                this.v1 = Scheduler.ID_HANDLER_B;
            } else {
                this.v1 = Scheduler.ID_HANDLER_A;
            }
            packet.id = this.v1;
            packet.a1 = 0;
            for (var i = 0; i < Packet.DATA_SIZE; i++) {
                this.v2++;
                if (this.v2 > 26) this.v2 = 1;
                packet.a2[i] = this.v2;
            }
            return this.scheduler.queue(packet);
        }
    }

    override function toString() : string {
        return "WorkerTask";
    }
}

class HandlerTask extends Task {

    var v1 : Packet;
    var v2 : Packet;

    /**
     * A task that manipulates work packets and then suspends itself.
     * @param scheduler the scheduler that manages this task
     * @constructor
     */
    function constructor(scheduler : Scheduler) {
        super(scheduler);
        this.v1 = null;
        this.v2 = null;
    }

    override function run(packet : Packet) : TaskControlBlock {
        if (packet != null) {
            if (packet.kind == Scheduler.KIND_WORK) {
                this.v1 = packet.addTo(this.v1);
            } else {
                this.v2 = packet.addTo(this.v2);
            }
        }
        if (this.v1 != null) {
            var count = this.v1.a1;
            var v;
            if (count < Packet.DATA_SIZE) {
                if (this.v2 != null) {
                    v = this.v2;
                    this.v2 = this.v2.link;
                    v.a1 = this.v1.a2[count];
                    this.v1.a1 = count + 1;
                    return this.scheduler.queue(v);
                }
            } else {
                v = this.v1;
                this.v1 = this.v1.link;
                return this.scheduler.queue(v);
            }
        }
        return this.scheduler.suspendCurrent();
    }

    override function toString() : string {
        return "HandlerTask";
    }
}

/* --- *
 * P a c k e t
 * --- */

class Packet {

    static const DATA_SIZE = 4;

    var link : Packet;
    var id : int;
    var kind : int;
    var a1 : int;
    var a2 : int[];

    /**
     * A simple package of data that is manipulated by the tasks.  The exact layout
     * of the payload data carried by a packet is not importaint, and neither is the
     * nature of the work performed on packets by the tasks.
     *
     * Besides carrying data, packets form linked lists and are hence used both as
     * data and worklists.
     * @param link the tail of the linked list of packets
     * @param id an ID for this packet
     * @param kind the type of this packet
     * @constructor
     */
    function constructor(link : Packet, id : int, kind : int) {
        this.link = link;
        this.id = id;
        this.kind = kind;
        this.a1 = 0;
        this.a2 = new Array.<int>(Packet.DATA_SIZE);
    }

    /**
     * Add this packet to the end of a worklist, and return the worklist.
     * @param queue the worklist to add this packet to
     */
    function addTo(queue : Packet) : Packet {
        this.link = null;
        if (queue == null) return this;
        var peek, next = queue;
        while ((peek = next.link) != null)
            next = peek;
        next.link = this;
        return queue;
    }

    override function toString() : string {
        return "Packet";
    }
}

// vim: set expandtab:
