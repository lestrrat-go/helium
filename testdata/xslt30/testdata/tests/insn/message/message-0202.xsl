<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.mytest.net"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="my"
             version="2.0">
<!-- Purpose: Tests calling a function in the select attribute of an xsl:message instruction.  -->

   <t:template match="docs">
   		<out>
         <t:message terminate="no" select="my:func()"/>
      </out>
	  </t:template>

   <t:function name="my:func">
		    <a xmlns:xsl="http://www.w3.org/1999/XSL/Transform">This is an error message.</a>
	  </t:function>
</t:transform>
