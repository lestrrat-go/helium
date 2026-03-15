<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:f="http://impincl32/ns" 
  xmlns:local="http://impincl32/ns/local" exclude-result-prefixes="f local">

<?spec xslt#apply-imports?>
  <!-- Creator: Michael Kay -->
  <!-- Purpose: See what happens when you try to circumvent the rule that
       "xsl:import must come first" by including a module that imports the target module -->

<xsl:variable name="v" select="0"/>

<xsl:template name="t">
  <xsl:value-of select="0"/>
</xsl:template>

<xsl:function name="f:f">
  <xsl:value-of select="0"/>
</xsl:function>

<xsl:template match="/">
  <xsl:value-of select="0"/>
</xsl:template>

<xsl:decimal-format grouping-separator="!"/>

<xsl:namespace-alias stylesheet-prefix="f" result-prefix="local"/>

<xsl:attribute-set name="a">
  <xsl:attribute name="a" select="0"/>
</xsl:attribute-set>

<xsl:output indent="no"/>

</xsl:stylesheet>