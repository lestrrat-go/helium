<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" exclude-result-prefixes="xs">
   <!-- Purpose: Test case to verify that in an xsl:messsage instruction Zero-length 
  				text nodes within the result sequence are removed. -->

   <xsl:template match="/">
      <xsl:variable name="zero-length-string" as="xs:string">
         <xsl:value-of select="''"/>
      </xsl:variable>
      <xsl:variable name="zero-length-text" as="text()">
         <xsl:text/>
      </xsl:variable>
      <out>
         <xsl:message select="'Error Message:'">
            <xsl:value-of
               select="('a', '', 'b', $zero-length-string, 'c', $zero-length-text,'d', no-such-node,'e')"
               separator="|"/>
         </xsl:message>
      </out>
   </xsl:template>

   <xsl:template match="text()"/>
</xsl:transform>
