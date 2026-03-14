<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#named-templates?>
  <!-- Purpose: Test for basic recursion on named "xsl:template". -->

<xsl:template name="template1">
 <xsl:param name="param1"></xsl:param>
  <xsl:if test = "$param1 = 1">
     First call <xsl:value-of select="$param1"/> 
   <xsl:call-template name="template1">
    <xsl:with-param name="param1">2</xsl:with-param>
   </xsl:call-template>
  </xsl:if>

  <xsl:if test = "$param1 = 2">
     Second call (recursing): <xsl:value-of select="$param1"/>
   <xsl:call-template name="template1">
     <xsl:with-param name="param1">3</xsl:with-param>
   </xsl:call-template>
   </xsl:if>

  <xsl:if test = "$param1 = 3">
     Third call (recursing): <xsl:value-of select = "$param1"/>
  </xsl:if>
</xsl:template>

<xsl:template match="doc">
 <out>
  <xsl:call-template name="template1">
    <xsl:with-param name="param1">1</xsl:with-param>
  </xsl:call-template>
 </out>
</xsl:template>
</xsl:stylesheet>
