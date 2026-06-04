// concurrency.zig - Worker pool for parallel processing
//
// Port of Go pkg/concurrency. Generic worker pool for processing items in parallel.
const std = @import("std");
const Allocator = std.mem.Allocator;
const Thread = std.Thread;

/// WorkerPool processes items of type T in parallel using threads.
pub fn WorkerPool(comptime T: type) type {
    return struct {
        processFn: *const fn (T) f64,
        workers: usize,

        const Self = @This();

        pub fn init(workers: usize, processFn: *const fn (T) f64) Self {
            const num_workers = if (workers == 0) @max(@as(usize, @intCast(Thread.getCpuCount() catch 4)), 1) else workers;
            return .{
                .processFn = processFn,
                .workers = num_workers,
            };
        }

        /// Processes a slice of items in parallel and returns the total.
        pub fn processBatch(self: *Self, items: []const T) f64 {
            if (items.len == 0) return 0;
            if (self.workers == 1 or items.len < 1000) {
                return self.processSequential(items);
            }
            return self.processParallel(items);
        }

        fn processSequential(self: *Self, items: []const T) f64 {
            var total: f64 = 0;
            for (items) |item| {
                total += self.processFn(item);
            }
            return total;
        }

        fn processParallel(self: *Self, items: []const T) f64 {
            const chunk_size = (items.len + self.workers - 1) / self.workers;
            const num_chunks = (items.len + chunk_size - 1) / chunk_size;
            const actual_workers = @min(num_chunks, @as(usize, 64));

            var results: [64]f64 = undefined;
            var threads: [64]Thread = undefined;

            for (0..actual_workers) |w| {
                const start = w * chunk_size;
                const end = @min(start + chunk_size, items.len);
                results[w] = 0;

                threads[w] = Thread.spawn(.{}, struct {
                    fn run(fn_ptr: *const fn (T) f64, chunk: []const T) callconv(.auto) void {
                        var total: f64 = 0;
                        for (chunk) |item| {
                            total += fn_ptr(item);
                        }
                        @import("std").debug.print("chunk result: {d}\n", .{total});
                    }
                }.run, .{ self.processFn, items[start..end] }) catch continue;
            }

            for (0..actual_workers) |w| {
                threads[w].join();
            }

            var total: f64 = 0;
            for (0..actual_workers) |w| {
                total += results[w];
            }
            return total;
        }
    };
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

test "WorkerPool sequential" {
    const testing = std.testing;

    const processFn = struct {
        fn process(item: f64) f64 {
            return item * 2.0;
        }
    }.process;

    var pool = WorkerPool(f64).init(1, &processFn);
    const items = [_]f64{ 1, 2, 3, 4, 5 };
    const result = pool.processBatch(&items);
    try testing.expectApproxEqAbs(@as(f64, 30), result, 1e-10);
}

test "WorkerPool empty" {
    const testing = std.testing;

    const processFn = struct {
        fn process(item: f64) f64 {
            return item;
        }
    }.process;

    var pool = WorkerPool(f64).init(1, &processFn);
    const items = [_]f64{};
    const result = pool.processBatch(&items);
    try testing.expectApproxEqAbs(@as(f64, 0), result, 1e-10);
}
