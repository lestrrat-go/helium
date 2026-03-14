<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                expand-text="yes"
                version="3.0">

 
   <xsl:mode on-no-match="shallow-copy"/>
   
   <xsl:template match="/">
      <xsl:variable name="in">
         <doc xsl:type="xs:integer">23</doc>
      </xsl:variable>
      <xsl:variable name="copied">
         <xsl:apply-templates select="$in/*"/>
      </xsl:variable>
      <out>{$copied/doc instance of element(doc, xs:integer)}</out>
   </xsl:template>

   <xsl:template match="text()[.='10']">
      <xsl:text>ten</xsl:text>
   </xsl:template>

</xsl:transform>
