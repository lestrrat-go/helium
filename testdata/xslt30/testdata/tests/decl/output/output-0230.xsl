<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
    
    <!-- Purpose: Invalid value for html-version parameter -->

    <t:output method="xhtml" html-version="five"
        doctype-system="http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd"
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
