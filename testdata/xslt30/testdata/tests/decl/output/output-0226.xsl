<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
    <!-- Purpose: Test for XHTML 5 elements in SVG namespaces, mixing default and prefixed namespaces -->
    <!-- Assumes resolution of bug 20261 -->

    <t:output method="xhtml" html-version="5.0"/>

    <t:template match="/">
        <html xmlns="http://www.w3.org/1999/xhtml">
            <head><title>Heading</title></head>
            <body>
                <div>
                    <s:svg xmlns:s="http://www.w3.org/2000/svg" version="1.1">
                        <s:circle cx="100" cy="50" r="40" stroke="black" 
                            stroke-width="2" fill="red"/>
                    </s:svg>
                </div>
            </body>
        </html>
    </t:template>
</t:transform>
