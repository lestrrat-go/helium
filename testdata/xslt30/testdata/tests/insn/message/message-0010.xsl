<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: xsl:message with a user-defined error code, supplied as a lexical QName <out/>  -->


   <t:param name="code" select="'XXXX8888'"/>
   <t:template name="main">
   	  <out>
         <t:message error-code="Q{{http://example.com/err}}{$code}" terminate="true">Terminated</t:message>
      </out>
   </t:template>
</t:transform>
