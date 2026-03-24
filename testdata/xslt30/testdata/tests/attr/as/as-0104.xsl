<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test with local xsl:param and @as=empty-sequence(). 
  				The values for  @select are () and empty sequence obtained at run time. -->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:template match="/doc">
      <xslt:param name="par1" select="()" as="empty-sequence()"/>
      <xslt:param name="par2" select="/doc/item" as="empty-sequence()"/>
      <out>
         <xslt:value-of select="$par1 instance of empty-sequence()"/>
         <xslt:value-of select="$par2 instance of empty-sequence()"/>
         <xslt:value-of select="$par1 instance of xs:string"/>
         <xslt:value-of select="$par2 instance of xs:string"/>
      </out>
   </xslt:template>
</xslt:transform>
