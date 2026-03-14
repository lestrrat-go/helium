<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test with global xsl:variable and @as=empty-sequence(). Value for  
  				@select are () and empty sequence obtained at run time. -->

   <xslt:variable name="var1" select="()" as="empty-sequence()"/>

   <xslt:variable name="var2" select="/doc/item" as="empty-sequence()"/>

   <xslt:template match="/doc">
      <out>
         <xslt:value-of select="$var1 instance of empty-sequence()"/>
         <xslt:value-of select="$var2 instance of empty-sequence()"/>
         <xslt:value-of select="$var1 instance of xs:string"/>
         <xslt:value-of select="$var2 instance of xs:string"/>
      </out>
   </xslt:template>
</xslt:transform>
