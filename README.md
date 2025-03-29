# Uni V2 Testing

stretching my go muscles

a program that accepts:

- uniswapv2: pool address (eth mainnet)
- input_token: input token
- output_token: output token
- amount: amount of input_token

based on the principles of how uniswap v2 works described here:
https://uniswapv3book.com/milestone_0/constant-function-market-maker.html

the trade function:

$$ (x+r\Delta x)(y-\Delta y)=k $$

1. There's a pool with some amount of token 0 ($x$) and some amount of token 1 ($y$)
2. When we buy token 1 for token 0, we give some amount of token 0 to the pool ($\Delta x$)
3. The pool gives us some amount of token 1 in exchange ($\Delta y$)
4. The pool also takes a small fee ($r=1-\text{swap fee}$) from the amount of token 0 we gave
5. The reserve of token 0 changes ($x+r\Delta x$), and the reserve of token 1 changes as well ($y-\Delta y$)
6. The product of updated reserves must still equal $k$

prices are always determined in terms of each other:

$$ P_x = \frac{y}{x} $$
$$ P_y =\frac{x}{y} $$

because of handy trade function, As you can see, we can derive ${\Delta x}$ and ${\Delta y}$ from it, like so:

$$\Delta y = \frac{y r \Delta x}{x + r \Delta x}$$

$$\Delta x = \frac{x \Delta y}{r(y - \Delta y)}$$

We can always find the output amount using the ${\Delta y}$ function (when we are selling a known amount of tokens).

### Things that can be improved

- pool address doesn't need to be passed, can just be token0 and token1 addresses
- should display symbols of tokens in output amounts
- fee is hardcoded to 0.3
