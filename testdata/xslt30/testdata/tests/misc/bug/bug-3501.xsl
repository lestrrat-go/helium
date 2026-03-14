<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet version="2.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:html="http://www.w3.org/1999/xhtml"
                xmlns="http://www.w3.org/1999/xhtml"
                exclude-result-prefixes="html">

<?spec xslt#patterns?>

<xsl:output method="xhtml"
            encoding="ISO-8859-1"
            indent="no" />

<xsl:template match="/">
  <xsl:copy>
    <xsl:apply-templates select="node()" />
  </xsl:copy>
</xsl:template>

<xsl:template match="@*|node()">
  <xsl:copy>
    <xsl:apply-templates select="@*|node()"/>
  </xsl:copy>
</xsl:template>

<xsl:template match="text()" priority="2">
  <xsl:value-of select="." />
</xsl:template>

<xsl:template match="html:h2">
  <h2>
    <a name="{.}">
      <xsl:value-of select="." />
    </a>
  </h2>
</xsl:template>

</xsl:stylesheet>

