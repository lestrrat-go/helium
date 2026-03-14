<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
exclude-result-prefixes="xs">

  <!-- Purpose: Test of pattern matching with For Expression inside predicate -->

<xsl:template match="/">
	<xsl:apply-templates select="//element()"/>
</xsl:template>

<xsl:template match="document-node()/*:doc/element()[for $t in * return $t/*]" priority="2">
  <t>
	<xsl:value-of select="name(.)"/>
  </t>
</xsl:template>

<xsl:template match="@*" priority="1"/>

<xsl:template match="processing-instruction()" priority="1"/>

<xsl:template match="element()" priority="1"/>

<xsl:template match="text()" priority="1"/>

<xsl:template match="comment()" priority="1"/>


</xsl:stylesheet>
