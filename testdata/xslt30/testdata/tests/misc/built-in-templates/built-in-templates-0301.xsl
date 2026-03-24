<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                expand-text="yes"
                version="3.0">

 
   <xsl:mode on-no-match="shallow-copy"/>
   
   <xsl:template match="/">
      <xsl:variable name="copied">
         <xsl:apply-templates/>
      </xsl:variable>
      <out>{$copied//empty instance of element(empty, xs:untyped)}</out>
   </xsl:template>

   <xsl:template match="sub-empty">
      <xsl:next-match/>
      <spanner-in-the-works/>
   </xsl:template>

</xsl:transform>
