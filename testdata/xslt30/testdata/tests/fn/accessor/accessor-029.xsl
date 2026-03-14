<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of fn:base-uri accessor with no argument  -->

   <xslt:template match="/doc">
	     <out>
         <t1>
            <xslt:value-of select="ends-with(base-uri(), 'accessor020.xml')"/>
         </t1>
         <t2>
            <xslt:for-each select="inner">
	              <xslt:value-of select="ends-with(base-uri(()), 'accessor020.xml')"/>
	           </xslt:for-each>
         </t2>
      </out>
	  </xslt:template>
</xslt:transform>
