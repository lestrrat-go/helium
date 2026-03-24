<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
    
    <!-- Purpose: Test for XHTML 5 elements with doctype-public and no doctype-system -->
    <!-- see bug 20264 -->

    <t:output method="xhtml" html-version="5.0"
        doctype-public="-//W3C//DTD XHTML 1.0 Strict//EN"
    />
 

    <t:template match="/">
        <html xmlns="http://www.w3.org/1999/xhtml">
            <head><title>Heading</title></head>
            <body>
                <p>Hello, world!</p>
            </body>
        </html>
    </t:template>
</t:transform>
