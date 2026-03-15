<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
    <!-- Purpose: Test for XHTML 5 elements in no namespace with non-empty content model.-->
    
    <t:output method="xhtml" html-version="5.0"/>
    
    <t:template match="/">
        <html xmlns="http://www.w3.org/1999/xhtml">
            <head>
                <title/>
            </head>
            <body>
                <p/>
                <i/>
                <u/>
                <div/>
                <code/>
                <strong/>
            </body>
        </html>
    </t:template>
</t:transform>
