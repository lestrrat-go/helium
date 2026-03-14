<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
    <!-- Purpose: Test for XHTML 5 DOCTYPE declaration with a top-level
         element in an alien namespace -->
    
    <t:output method="xhtml" html-version="5.0"/>
    
    <t:template match="/">
            <html xmlns="http://www.example.com/not-xhtml">
                <head>
                    <title>Not XHTML!</title>
                </head>
                <body>
                    <p>But what is it then?</p>
                </body>
            </html>
    </t:template>
</t:transform>
