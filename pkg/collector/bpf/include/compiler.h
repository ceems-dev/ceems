/* SPDX-License-Identifier: (GPL-3.0-only) */

/**
 * Seems like LRU hash map with fewer max entries have unexpected
 * behaviour.
 * Ref: https://stackoverflow.com/questions/75882443/elements-incorrectly-evicted-from-ebpf-lru-hash-map
 * 
 * We noticed in rudimentary tests as well where values are being
 * evicted even before map is full. So we use bigger maps to
 * ensure that we get a more LRUish behaviour in production.
 * 
*/
#ifndef MAX_MAP_ENTRIES
#define MAX_MAP_ENTRIES 16384
#endif

#ifndef MAX_MOUNT_SIZE
#define MAX_MOUNT_SIZE 64
#endif

#define FUNC_INLINE static inline __attribute__((always_inline))

/*
 * Following define is to assist VSCode Intellisense so that it treats
 * __builtin_preserve_access_index() as a const void * instead of a
 * simple void (because it doesn't have a definition for it). This stops
 * Intellisense marking all _(P) macros (used in probe_read()) as errors.
 * To use this, just define VSCODE in 'C/C++: Edit Configurations (JSON)'
 * in the Command Palette in VSCODE (F1 or View->Command Palette...):
 *    "defines": ["VSCODE"]
 * under configurations.
 */
#ifdef VSCODE
const void *__builtin_preserve_access_index(void *);
#endif
#define _(P) (__builtin_preserve_access_index(P))

#ifndef likely
#define likely(X) __builtin_expect(!!(X), 1)
#endif

#ifndef unlikely
#define unlikely(X) __builtin_expect(!!(X), 0)
#endif

#ifndef __inline__
#define __inline__ __attribute__((always_inline))
#endif

#define DEBUG
#ifdef DEBUG
/* Only use this for debug output. Notice output from bpf_trace_printk()
 * ends up in /sys/kernel/debug/tracing/trace_pipe
 */
#define bpf_debug(fmt, ...)                                                \
	({                                                                 \
		char ____fmt[] = fmt;                                      \
		bpf_trace_printk(____fmt, sizeof(____fmt), ##__VA_ARGS__); \
	})
#else
#define bpf_debug(fmt, ...) \
	{                   \
		;           \
	}
#endif

// Just to ensure that we can use vfs_write/vfs_read calls
// Picked from https://github.com/torvalds/linux/blob/master/tools/include/linux/types.h#L56
#ifndef __user
#define __user
#endif

/*
 *   gcc: https://gcc.gnu.org/onlinedocs/gcc/Common-Function-Attributes.html#index-warn_005funused_005fresult-function-attribute
 * clang: https://clang.llvm.org/docs/AttributeReference.html#nodiscard-warn-unused-result
 */
#define __must_check __attribute__((__warn_unused_result__))

#ifndef __force
#define __force
#endif
/*
 * Kernel pointers have redundant information, so we can use a
 * scheme where we can return either an error code or a normal
 * pointer with the same return value.
 *
 * This should be a per-architecture thing, to allow different
 * error and pointer decisions.
 */
#define MAX_ERRNO 4095

/**
 * IS_ERR_VALUE - Detect an error pointer.
 * @x: The pointer to check.
 *
 * Like IS_ERR(), but does not generate a compiler warning if result is unused.
 *
 * Fetched from kernel source https://github.com/torvalds/linux/blob/master/include/linux/err.h#L28
 */
#define IS_ERR_VALUE(x) unlikely((unsigned long)(void *)(x) >= (unsigned long)-MAX_ERRNO)

/**
 * IS_ERR - Detect an error pointer.
 * @ptr: The pointer to check.
 * Return: true if @ptr is an error pointer, false otherwise.
 */
FUNC_INLINE bool __must_check IS_ERR(__force const void *ptr)
{
	return IS_ERR_VALUE((unsigned long)ptr);
}
