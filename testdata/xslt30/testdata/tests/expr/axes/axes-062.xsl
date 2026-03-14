<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: Test for 'ancestor::' Axis Identifier, with XPath 2.0 unordered() function -->

  <xsl:template match="/">
    <out>
      <ordered>
          <xsl:for-each select="//center">
            <xsl:apply-templates select="ancestor::*"/>
          </xsl:for-each>
      </ordered>
      <unordered>
         <xsl:variable name="unordered-ancestors" as="text()*">
          <xsl:for-each select="//center">
            <xsl:apply-templates select="unordered(ancestor::*)"/>
          </xsl:for-each>
         </xsl:variable>
         <!-- make the results predictable -->
         <xsl:perform-sort select="$unordered-ancestors">
           <xsl:sort select="string-length(.)"/>
         </xsl:perform-sort>
      </unordered>      
  	</out>
  </xsl:template>
    
  <xsl:template match="*">
     <xsl:value-of select="name(.)"/><xsl:text> </xsl:text>
  </xsl:template>
 
</xsl:stylesheet>
