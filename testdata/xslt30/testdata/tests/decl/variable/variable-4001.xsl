<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#with-param?>
    <!-- Purpose: Test for passing param containing boolean via apply-templates -->

<xsl:output method="xml" encoding="UTF-8"/>

<xsl:template match="doc">
  <out>
    <xsl:apply-templates select="a">
      <xsl:with-param name="eq" select="a=b"/>
    </xsl:apply-templates>
  </out>
</xsl:template>

<xsl:template match="a">
  <xsl:param name="eq">Default text in param 1</xsl:param>
  <a>
    <xsl:value-of select="."/>
  </a>
  <b>
    <xsl:if test="$eq">
      <xsl:attribute name="content">
        <xsl:value-of select="'the same'"/>
      </xsl:attribute>
    </xsl:if>
    <xsl:value-of select="."/>
  </b>
</xsl:template>

</xsl:stylesheet>