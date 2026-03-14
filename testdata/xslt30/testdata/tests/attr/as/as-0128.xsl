<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://www.mytest.net"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs my"
                version="2.0">
<!-- Purpose: Test type of result of xsl:function that has @as="attribute() 
  				and sequence constructor contains xsl:attribute.-->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:function name="my:func1" as="attribute()">
	     <xslt:attribute name="attr">red</xslt:attribute>
   </xslt:function>

   <xslt:template match="/doc">
      <out>
         <xslt:value-of select="my:func1() instance of attribute()"/>
      </out>
   </xslt:template>
</xslt:transform>
