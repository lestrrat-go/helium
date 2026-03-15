<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
    <!-- Purpose: Test for XHTML 5 elements in no namespace with non-empty content model,
        this time with attributes.-->
    
    <t:output method="xhtml" html-version="5.0"/>
    
    <t:template match="/">
        <html>
            <head>
                <title class="c"/>
            </head>
            <body>
                <p class="c"/>
                <i class="c"/>
                <u class="c"/>
                <div class="c"/>
                <code class="c"/>
                <strong class="c"/>
            </body>
        </html>
    </t:template>
</t:transform>
