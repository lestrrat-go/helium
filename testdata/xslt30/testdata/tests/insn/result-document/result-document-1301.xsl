<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
    <!-- Purpose: Check expression 'position() eq last()' is evaluated correctly when producing
    multiple result documents-->
    <t:template match="/">
        <t:for-each select="//section">
            <t:result-document method="html" href="section{position()}.html">
                <html>
                    <body>
                        <h1>Section <t:value-of select="concat(position(),' of ', last())"/></h1>
                        <p>position() eq last() = <t:value-of select="position() eq last()"/></p>
                        <p>position() ne last() = <t:value-of select="position() ne last()"/></p>
                    </body>
                </html>
            </t:result-document>
        </t:for-each>
    </t:template>
</t:transform>