<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" xmlns:f="http://f/">

<?spec xslt#attribute-value-templates?>
  <!-- PURPOSE: Test handling of text nodes in an AVT. -->
  <!-- AUTHOR: Michael Kay -->
  <xsl:template match="/">
    <out test="{f:t(0)}{f:t(1)}{f:t(2)}"/>
  </xsl:template>
  
  <xsl:function name="f:t" as="text()*">
   <xsl:param name="p"/>
   <xsl:text/>[<xsl:value-of select="$p"/>]<xsl:text/>
  </xsl:function>
 
</xsl:stylesheet>
