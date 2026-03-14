<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#stylesheet-stripping?>
    <!-- Purpose: Processing Instructions and comments within a stylesheet 
       are ignored. -->

<?MyPI version="1.0"?>

<xsl:template match="doc">
  <out>
  <?MyPI version="2.0"?>
  <!-- This template prints out 'Testing 1 2 3' -->
    <xsl:text>&#010;</xsl:text>
    <xsl:value-of select="'Testing '"/>
    <xsl:for-each select="*">
      <xsl:value-of select="."/><xsl:text> </xsl:text>		
      <?MyPI version="3.0"?>
    </xsl:for-each>

  <!-- This calls a named template -->
    <xsl:call-template name="ThatTemp">
      <xsl:with-param name="sam">quos</xsl:with-param>
      <?MyPI version="4.0"?>
    </xsl:call-template>
    <xsl:text>&#010;</xsl:text>
  </out>
</xsl:template>

<!-- This is the named template that is called from above -->
<xsl:template name="ThatTemp">
  <xsl:param name="sam">bo</xsl:param>
  <?MyPI version="5.0"?>
  <xsl:copy-of select="$sam"/>
</xsl:template>

</xsl:stylesheet>
