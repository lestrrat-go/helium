<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
    <!-- Purpose: Test for XHTML 5 elements in XHTML namespace with empty content model.-->
    
    <t:output method="xhtml" html-version="5.0"/>
    
    <t:template match="/">
        <html xmlns="http://www.w3.org/1999/xhtml">
            <head>
                <title>test</title>
            </head>
            <body>
                <area/>
                <base/>
                <br/>
                <col/>
                <embed/>
                <hr/>
                <img/>
                <input/>
                <link/>
                <meta/>
                <param/>
                <source/>
                <track/>
                <wbr/>
            </body>
        </html>
    </t:template>
</t:transform>
