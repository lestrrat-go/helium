<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#patterns?>
  <!-- Purpose: Show that a variable can be used in a match pattern, though
      not for the name test. The variable must be top-level, of course. -->
  
<xsl:variable name="screen" select="7"/>

<xsl:template match="/doc">
  <out>
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template match="foo[. &gt; $screen]">
  <xsl:text>Passed: </xsl:text><xsl:value-of select="@att1"/>
</xsl:template>

<xsl:template match="*">
  <xsl:text>Failed: </xsl:text><xsl:value-of select="@att1"/>
</xsl:template>

</xsl:stylesheet>
