# Math 

## Part I: Calculus for Data Science

### Section 1: Calculus I (Single Variable)

#### Lesson 1: Introduction

    # 1. Boundaries

    - Since our objective is calculus for Data Science, we will focus only on
      real numbers.

    # 2. What is calculus?

    Imagine, you are driving your car. One question, you may ask is what is
    your average speed during your drive. 

        v_avg = d / t

    But during that drive you had different speeds - you were fast on highways,
    slow at traffic signals. So, there is this experiential notion of
    'instantaneous' speed i.e. what is your speed at any given moment? 

    Calculus is a framework that gives us the tools to help us define a precise
    notion for that 'instantaneous rate of change'.

    # 3. What does calculus have to do with data science?

    - Calculus 1 (Derivatives): While speed may seem related to 'physics', in 
      mathematics it is just a slope in a 'distance v. time' chart. And calculus 
      is all about finding such slopes. This is Calculus 1.

    - Calculus 2 (Integerals): Calculates totals or areas under curves, like 
      summing up probabilities in stats or total errors in models. It's used for 
      things like predicting averages or evaluating ML accuracy.

    - Calculus 3 (Multivariable): In multiple dimensions, the slope is analogous 
      to the gradient, which tells us more about the steepest ascent and the 
      steepest descent. Gradient descents are key to how machine learning 
      algorithms like neural networks actually learn. 

#### Lesson 2: Functions

    # 1. What are functions?

    Essentially, functions are 'mappings'/'rules' that assign each input x in a 
    domain to exactly one output f(x) in a co-domain. 

    The easiest way to represent a function is a two column table, of input and
    output.

    #! | input_domain | output_co_domain |
    #! -----------------------------------
    #! |            1 |                1 |
    #! |            2 |                4 |
    #! |            3 |                9 |
    #! -----------------------------------

    But, the problem with the above representation is that it is not very
    useful for calculus. For this reason, we represent the same function above
    as below, as it gives us
    - better abstraction
    - a better way to visualize them while coding

        f(x) = x^2
        g(x) = pi*x

    - the above functions may also be represented as below,

        y = x^2
        y = pi*x


    # 2. What types of functions are there?

    ## 2.1. Straight Lines

        f(x) = y = m*x + b

    ## 2.2. Polynomials (Curvy lines)

        f(x) = y = a*x^3 + b*x^2 + c*x + d  
        # The below root functions are also polynomials
        f(x) = y = a*x^(1/3) + b*x^(1/2) + c

    ## 2.3. Exponential Functions

    Here, x appears in the exponent. Such functions grow very 'steeply' when
    plotted, as their growth is 'exponential'.

        f(x) = y = 2^x

    In calculus we will learn of a very special number called e, and it is very
    common to see functions like this:

        f(x) = y = e^x

    ## 2.4. Logarithmic Functions

    ### 2.4.1. Why do log and ln even exist? 
    
    - log: In the late 1500s, multiplying two 8-digit numbers by hand took a long 
    time and is very error-prone. Doing dozens or hundreds of such operations per 
    day (common in astronomy tables or navigation) was exhausting. John Napier 
    (Scottish baron, mathematician, and tinkerer) had the breakthrough insight:
    Turn troublesome multiplication and division into easy addition and 
    subtraction. He spent ~20 years developing this. He defined a function log such 
    that, for the same 'base' x:

        log_{x}(a*b) = log_{x}(a) + log_{x}(b)

    - ln: It exists because it makes it easy to find the time (or amount of input) 
    needed for exponential growth/decay to reach a certain level, and also directly
    gives the area under the curve.

    ### 2.4.2. What are log and ln?

    When you take a log, it is with respect to a base. So, if we simply say
    log(x) without any base qualifier, we mean base 10.

        f(x) = y = log(x) = log_{10}(x)

    Likewise, when we take a ln (pronounced lawn), without specifying a base,
    we mean the log of x with base e.

        f(x) = y = ln(x) = log_{e}(x)

    But, generally speaking, we can have any base b of x.

        f(x) = y = log_{b}(x)

    One useful thing to review is how we can convert one base to another. This
    is useful, because in calculus, it is usually more convenient to use to ln
    instead of the log. So, we will probably end up doing things like this: 

        log_b(x) = log_c(x) / log_c(b) = ln(x) / ln(b)

    It is worth noting that while convering advanced courses on statistics or
    probabilities, most people say 'log', even though they mean 'ln' - we don't 
    know what that is the case - but it is what it is.

    ## 2.5. Trigonometric Functions

    These are the three most useful trigonometric functions.

        y = sin(x)
        y = cos(x)
        y = tan(x) = sin(x) / cos(x)

    These functions come from a right angled triangle, with angle theta, and
    sides a, o, and h (representing the sides adjacent theta, opposite theta, and
    the hypotenuse). In this context,

        sin(theta) = o/h
        cos(theta) = a/h
        tan(theta) = sin(theta)/cos(theta) = (o/h)/(a/h) = o/a

    ## 2.6. Hyperbolic Functions

    Hyperbolic functions have this notation:

        y = sinh(x)
        y = cosh(x)
        y = tanh(x) = sinh(x)/cosh(h)

    So something unique about the tanh(x) which is unique to machine learning
    is that it has an output range of (-1, 1), so it is zero-centered and often
    used as an activation function in neural networks.

    So, it turns out that we can express the hyperbolic functions in terms of
    exponential functions.

        sinh(x) = (e^x - e^(-x))/2
        cosh(x) = (e^x + e^(-x))/2
        tanh(x) = (e^x - e^(-x))/(e^x + e^(-x))

    The above us useful because, when we are taking derivatives, we will know
    how to take the derivative of e^x, but we wont know directly how to take
    the derivative of tanh(x).

    ## 2.7. Ratios of Functions

    One thing to note, is that we can have ratios of functions. For instance,
    the tanh(x) function expands to a ratio of exponential functions.

        tanh(x) = (e^x - e^(-x))/(e^x + e^(-x))

    Likewise, we can have have,

        f(x) = (x^2 - 2x + 3)/(x^5 + 2x^3 + 1)

    One important function in signal processing/ timeseries analysis, is

        f(x) = sin(x)/x
    
    ## 2.8. Expressing Functions in Python

    import numpy as np
    import matplotlib.pyplot as plt


    def plot_func(file_name, x, *y_funcs):
        fig, ax = plt.subplots()
        ax.axhline(y=0, color="k")
        ax.axvline(x=0, color="k")
        ax.grid(True)

        for y in y_funcs:
            ax.plot(x, y)

        fig.savefig(file_name, dpi=200, bbox_inches="tight")
        plt.close(fig)


    # Plotting Linear Polynomial func
    x = np.linspace(-5, 5, 100)
    y = 0.5 * x + 1
    plot_func("plot1.png", x, y)

    # Plotting Quadratic funcs
    x = np.linspace(-2, 2, 100)
    y = -0.5 * 9.8 * x ** 2 + 2 * x + 1
    plot_func("plot2.png", x, y)

    x = np.linspace(0,10,100)
    y = -0.5 * x ** (1/3) + 0.2 * x ** (1/2) + 1
    plot_func("plot3.png", x, y)
    # Plotting Exponential funcs
    x = np.linspace(-5,5,100)
    y = np.e ** x # This is the same the magic number 'e' raised to the power of x
    plot_func("plot4.png", x, y)
    # In so far as machine learning is concerned, we need to remember that
    # exponential functions grow much faster than linear or quadratic functions.

    # Here, we are using the log function, to represent ln(x). In the numpy
    # library, when you can the log function in relation to x, it actually means
    # ln(x) - it is base e, not base 10 by default.
    x = np.linspace(0.0000001, 10, 200)
    y = np.log(x)
    plot_func("plot5.png", x, y)

    # Now, plotting trigonometric functions.
    x = np.linspace(-2 * np.pi, 2 * np.pi, 200)
    y1 = np.sin(x)
    y2 = np.cos(x)
    plot_func("plot6.png", x, y1, y2)
    # From this plot, we can see that we may think of the cosine as a shifted
    # version of the sin.

    [At 07:30,
    https://www.udemy.com/course/calculus-data-science/learn/lecture/36251162#content]


#### Lesson 1: What is calculus?

### Section 2: Calculus II (Integration)

#### Lesson 1: 

### Section 3: Calculus III (Vector/ Multivariable)

#### Lesson 1: 






