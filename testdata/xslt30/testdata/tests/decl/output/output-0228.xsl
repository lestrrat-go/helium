<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
    
    <!-- Purpose: Test for XHTML 5 elements with doctype-system and no doctype-public -->
    <!-- Also checks that "+5.0" is legal for the version number -->

    <t:output method="xhtml" html-version="+5.0"
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
