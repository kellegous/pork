// todo: export this
#define DEBUG

#if defined(DEBUG)
#define ASSERT(COND) window.console.assert(COND, #COND)
#else
#define ASSERT(cond)
#endif