<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
    <!-- Purpose: Test for XHTML 5 elements in SVG and MathML namespaces, used as default namespace -->

    <t:output method="xhtml" html-version="5.0"/>

    <t:template match="/">
        <h:html xmlns:h="http://www.w3.org/1999/xhtml">
            <h:head>
                <h:title>test</h:title>
            </h:head>
            <h:body>
                <h:div>
                    <s:svg xmlns:s="http://www.w3.org/2000/svg" version="1.1">
                        <s:circle cx="100" cy="50" r="40" stroke="black" stroke-width="2" fill="red"/>
                    </s:svg>
                </h:div>
                <h:div>
                    <m:math display="block" xmlns:m="http://www.w3.org/1998/Math/MathML">
                        <m:mrow>
                            <m:mi>cos</m:mi>
                            <m:mo>&#x2061;</m:mo>
                            <m:mrow>
                                <m:mo>(</m:mo>
                                <m:mi>&#x03b8;</m:mi>
                                <m:mo>+</m:mo>
                                <m:mi>&#x03c6;</m:mi>
                                <m:mo>)</m:mo>
                            </m:mrow>
                            <m:mo>=</m:mo>
                            <m:mi>cos</m:mi>
                            <m:mo>&#x2061;</m:mo>
                            <m:mrow>
                                <m:mo>(</m:mo>
                                <m:mi>&#x03b8;</m:mi>
                                <m:mo>)</m:mo>
                            </m:mrow>
                            <m:mi>cos</m:mi>
                            <m:mo>&#x2061;</m:mo>
                            <m:mrow>
                                <m:mo>(</m:mo>
                                <m:mi>&#x03c6;</m:mi>
                                <m:mo>)</m:mo>
                            </m:mrow>
                            <m:mo>&#x2212;</m:mo>
                            <m:mi>sin</m:mi>
                            <m:mo>&#x2061;</m:mo>
                            <m:mrow>
                                <m:mo>(</m:mo>
                                <m:mi>&#x03b8;</m:mi>
                                <m:mo>)</m:mo>
                            </m:mrow>
                            <m:mi>sin</m:mi>
                            <m:mo>&#x2061;</m:mo>
                            <m:mrow>
                                <m:mo>(</m:mo>
                                <m:mi>&#x03c6;</m:mi>
                                <m:mo>)</m:mo>
                            </m:mrow>
                        </m:mrow>
                    </m:math>
                </h:div>
            </h:body>
        </h:html>
    </t:template>
</t:transform>
