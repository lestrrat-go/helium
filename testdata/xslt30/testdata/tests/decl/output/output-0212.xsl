<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
    <!-- Purpose: Test for XHTML 5 DOCTYPE declaration.
         Uses "5.00" rather than the more usual "5.0" 
         Uses whitespace before the html element.
    -->
    
    <t:output method="xhtml" html-version="5.00"/>
    
    <t:template match="/">
        <t:text>   </t:text>
        <html xmlns="http://www.w3.org/1999/xhtml">
            <head>
                <title>test</title>
            </head>
            <body>
                <p>content</p>
            </body>
        </html>
    </t:template>
</t:transform>
