<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
    <!-- Purpose: Test for XHTML 5 DOCTYPE declaration with a namespace prefix.
         Top element is not html so there is no DOCTYPE declaration -->
    
    <t:output method="xhtml" html-version="5.0"/>
    
    <t:template match="/">
            <h:body xmlns:h="http://www.w3.org/1999/xhtml">
                <h:p>content</h:p>
            </h:body>
    </t:template>
</t:transform>
