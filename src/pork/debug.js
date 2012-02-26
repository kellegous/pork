// todo: export this
#define DEBUG

#if defined(DEBUG)
#define ASSERT(c, s) console.assert(c, s);
#else
#define ASSERT(c, s)
#endif