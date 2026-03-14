<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
   <!-- Purpose: Test terminate="true" attribute on xsl:message element. -->

   <t:variable name="v">
        <a xmlns:xsl="http://www.w3.org/1999/XSL/Transform">boo</a>
    </t:variable>

   <t:template name="main">
   		<out>
         <t:message terminate="true">Yah boo</t:message>
      </out>
	  </t:template>
</t:transform>
