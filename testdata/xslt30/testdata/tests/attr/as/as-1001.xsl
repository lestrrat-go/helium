<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test with an xs:anyURI value as a child of LRE in the sequence constructor of a non-tunnel xsl:with-param and @as=xs:anyURI+. 
  				Verify the parameter is of type xs:anyURI+. -->

   <xslt:template match="/">
      <out>
         <xslt:apply-templates>
	           <xslt:with-param name="par1" as="xs:anyURI+" tunnel="no">
		             <elem>
                  <xslt:value-of select="xs:anyURI('http://test.org')"/>
               </elem>
	           </xslt:with-param>
         </xslt:apply-templates>
      </out>
   </xslt:template>

   <xslt:template match="item">
      <xslt:param name="par1" select="'1.0'"/>
      <par1>
         <xslt:value-of select="$par1"/>
         <xslt:value-of select="$par1 instance of xs:anyURI+"/>
         <xslt:value-of select="$par1 instance of xs:anyURI"/>
      </par1>
   </xslt:template>
</xslt:transform>
