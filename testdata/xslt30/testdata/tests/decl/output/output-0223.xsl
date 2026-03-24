<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
    <!-- Purpose: Test for XHTML 5 elements in no namespace with empty content model,
         this time with attributes -->
    
    <t:output method="xhtml" html-version="5.0"/>
    
    <t:template match="/">
        <html>
            <head>
                <title>test</title>
            </head>
            <body>
                <area class="s"/>
                <base class="s"/>
                <br class="s"/>
                <col class="s"/>
                <embed class="s"/>
                <hr class="s"/>
                <img class="s"/>
                <input class="s"/>
                <link class="s"/>
                <meta class="s"/>
                <param class="s"/>
                <source class="s"/>
                <track class="s"/>
                <wbr class="s"/>
            </body>
        </html>
    </t:template>
</t:transform>
