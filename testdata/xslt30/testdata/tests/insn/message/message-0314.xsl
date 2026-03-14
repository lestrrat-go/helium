<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.mytest.net"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="my"
             version="2.0">
<!-- Purpose: Test case that uses doc() in the select attribute of an xsl:message instruction (The message text is retrieved from an xml file).  -->

   <t:template match="/">
   		<out>
         <t:message select="doc('errorMessage.xml')/outer/error/text()"/>
      </out>
	  </t:template>
</t:transform>
