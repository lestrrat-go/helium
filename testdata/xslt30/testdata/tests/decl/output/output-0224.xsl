<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
    <!-- Purpose: Test for XHTML 5 elements in SVG and MathML namespaces, used as default namespace -->

    <t:output method="xhtml" html-version="5.0"/>

    <t:template match="/">
        <html xmlns="http://www.w3.org/1999/xhtml">
            <head>
                <title>test</title>
            </head>
            <body>
                <div>
                    <svg xmlns="http://www.w3.org/2000/svg" version="1.1">
                        <circle cx="100" cy="50" r="40" stroke="black" stroke-width="2" fill="red"/>
                    </svg>
                </div>
                <div>
                    <math display="block" xmlns="http://www.w3.org/1998/Math/MathML">
                        <mrow>
                            <mi>cos</mi>
                            <mo>&#x2061;</mo>
                            <mrow>
                                <mo>(</mo>
                                <mi>&#x03b8;</mi>
                                <mo>+</mo>
                                <mi>&#x03c6;</mi>
                                <mo>)</mo>
                            </mrow>
                            <mo>=</mo>
                            <mi>cos</mi>
                            <mo>&#x2061;</mo>
                            <mrow>
                                <mo>(</mo>
                                <mi>&#x03b8;</mi>
                                <mo>)</mo>
                            </mrow>
                            <mi>cos</mi>
                            <mo>&#x2061;</mo>
                            <mrow>
                                <mo>(</mo>
                                <mi>&#x03c6;</mi>
                                <mo>)</mo>
                            </mrow>
                            <mo>&#x2212;</mo>
                            <mi>sin</mi>
                            <mo>&#x2061;</mo>
                            <mrow>
                                <mo>(</mo>
                                <mi>&#x03b8;</mi>
                                <mo>)</mo>
                            </mrow>
                            <mi>sin</mi>
                            <mo>&#x2061;</mo>
                            <mrow>
                                <mo>(</mo>
                                <mi>&#x03c6;</mi>
                                <mo>)</mo>
                            </mrow>
                        </mrow>
                    </math>
                </div>
            </body>
        </html>
    </t:template>
</t:transform>
