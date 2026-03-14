<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#conflict?>
<!-- Test for conflict Resolution  non_simple ('/').
     No conflict warnings should be seen.
	       
The spec states:
  " The idea is that the most common kind of pattern (a pattern that justs tests for
    an element with a specific name) has priority 0; a pattern more specific than
	this has priority .5; a pattern less specific than this has priority minus .5"

  -->

<xsl:template match="doc">
  <out>
    <xsl:apply-templates select="foo"/>
  </out>
</xsl:template>
  

<xsl:template match="doc/foo">
    <xsl:text>Match of non-simple '/'</xsl:text>
</xsl:template>

<xsl:template match="foo">
    <xsl:text>Match-of-qualified-name</xsl:text>
</xsl:template>

<xsl:template match="*">
    <xsl:text>Match-of-wildcard</xsl:text>
</xsl:template>

<xsl:template match="node()">
    <xsl:text>Match-of-node-type</xsl:text>
</xsl:template>

</xsl:stylesheet>
