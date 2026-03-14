<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
    <!-- Purpose: Test for XHTML 5 elements in XHTML namespace with non-empty content model,
        this time with attributes.-->
    
    <t:output method=" xhtml " html-version=" 5.0 "/>
    
    <t:template match="/">
        <h:html xmlns:h="http://www.w3.org/1999/xhtml">
            <h:head>
                <h:title class="c"/>
            </h:head>
            <h:body>
                <h:p class="c"/>
                <h:i class="c"/>
                <h:u class="c"/>
                <h:div class="c"/>
                <h:code class="c"/>
                <h:strong class="c"/>
            </h:body>
        </h:html>
    </t:template>
</t:transform>
