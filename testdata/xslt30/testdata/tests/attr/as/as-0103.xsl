<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://www.myfunctions.org"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs my"
                version="2.0">
<!-- Purpose: Test with xsl:function and @as=empty-sequence(). 
  				Use xsl:sequence with @select=() inside the sequence constructor. -->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:function name="my:func" as="empty-sequence()">
	     <xslt:sequence select="()"/>
   </xslt:function>

   <xslt:template match="/doc">
      <out>
         <xslt:value-of select="my:func()"/>
      </out>
   </xslt:template>
</xslt:transform>
